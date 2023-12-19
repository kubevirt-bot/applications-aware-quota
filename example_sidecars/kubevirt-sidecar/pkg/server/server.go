package server

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apiserver/pkg/quota/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/quota/v1/evaluator/core"
	"k8s.io/utils/clock"
	v15 "kubevirt.io/api/core/v1"
	"kubevirt.io/applications-aware-quota/pkg/client"
	"kubevirt.io/applications-aware-quota/pkg/log"
)

const (
	launcherLabel = "virt-launcher"
	// VmiPodUsage Calculate usage of launcher like any other pod but hide migration additional resources
	VmiPodUsage VmiCalcConfigName = "VmiPodUsage"
	// VirtualResources Calculate memory.request/limits as the vmi's ram size and cpu.request/limits as number of threads of vmi
	VirtualResources VmiCalcConfigName = "VirtualResources"
	// DedicatedVirtualResources Calculate vmi.requests.memory as the vmi's ram size and vmi.requests.cpu as number of threads of vmi
	// in this configuration no memory.request/limits and cpu.request/limits won't be included
	DedicatedVirtualResources VmiCalcConfigName = "DedicatedVirtualResources"
	DefaultLauncherConfig                       = "VmiPodUsage"
	//VirtualResources:
	// ResourcePodsOfVmi Launcher Pods, number.
	ResourcePodsOfVmi corev1.ResourceName = "requests.instances/vmi"
	// Vmi CPUs, Total Threads number(Cores*Sockets*Threads).
	ResourceRequestsVmiCPU corev1.ResourceName = "requests.cpu/vmi"
	// Vmi Memory Ram Size, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	ResourceRequestsVmiMemory corev1.ResourceName = "requests.memory/vmi"
)

type Server struct {
	virtLauncherCalculator *VirtLauncherCalculator
}

func (s *Server) PodUsageFunc(_ context.Context, request *PodUsageRequest) (*PodUsageResponse, error) {
	pod := &corev1.Pod{}
	var podsState []corev1.Pod
	err := json.Unmarshal(request.Pod.GetPodJson(), pod)
	if err != nil {
		return nil, err
	}
	for _, pItem := range request.GetPodsState() {
		currPod := &corev1.Pod{}
		err := json.Unmarshal(pItem.GetPodJson(), currPod)
		if err != nil {
			return nil, err
		}
		podsState = append(podsState, *currPod)
	}

	rl, err, match := s.virtLauncherCalculator.PodUsageFunc(*pod, podsState, clock.RealClock{})
	rlData, err := json.Marshal(rl)
	if err != nil {
		return nil, err
	}
	podUsageResponse := &PodUsageResponse{Error: &Error{false, ""}, Match: match, ResourceList: &ResourceList{rlData}}
	if err != nil {
		podUsageResponse.Error.Error = true
		podUsageResponse.Error.ErrorMessage = err.Error()
	}

	return podUsageResponse, nil
}

func (s *Server) HealthCheck(_ context.Context, _ *HealthCheckRequest) (*HealthCheckResponse, error) {
	return &HealthCheckResponse{true}, nil
}

type VmiCalcConfigName string

var MyConfigs = []VmiCalcConfigName{VmiPodUsage, VirtualResources, DedicatedVirtualResources}

func NewVirtLauncherCalculator(config string) *VirtLauncherCalculator {
	aaqCli, err := client.GetAAQClient()
	if err != nil {
		panic("NewVirtLauncherCalculator: couldn't get aaqCli")
	}
	return &VirtLauncherCalculator{
		aaqCli:     aaqCli,
		calcConfig: VmiCalcConfigName(config),
	}
}

type VirtLauncherCalculator struct {
	aaqCli     client.AAQClient
	calcConfig VmiCalcConfigName
}

func (launchercalc *VirtLauncherCalculator) PodUsageFunc(pod corev1.Pod, items []corev1.Pod, clock clock.Clock) (corev1.ResourceList, error, bool) {
	if pod.OwnerReferences == nil || len(pod.OwnerReferences) == 0 || pod.OwnerReferences[0].Kind != v15.VirtualMachineInstanceGroupVersionKind.Kind || !core.QuotaV1Pod(&pod, clock) {
		log.Log.Infof("doesn't match pod.name: %v\n", pod.Name)
		return corev1.ResourceList{}, nil, false
	}

	vmi, err := launchercalc.aaqCli.KubevirtClient().KubevirtV1().VirtualMachineInstances(pod.Namespace).Get(context.Background(), pod.OwnerReferences[0].Name, metav1.GetOptions{})
	if err != nil {
		log.Log.Infof("couldn't get vmi %v err: %v\n", pod.OwnerReferences[0].Name, err.Error())
		return corev1.ResourceList{}, fmt.Errorf("couldn't get vmi %v err: %v", pod.OwnerReferences[0].Name, err.Error()), false
	}

	launcherPods := unfinishedVMIPods(launchercalc.aaqCli, items, vmi)

	if !podExists(launcherPods, &pod) { //sanity check
		log.Log.Infof("can't detect pod as launcher pod pod.name: %v\n", pod.Name)
		return corev1.ResourceList{}, fmt.Errorf("can't detect pod as launcher pod"), true
	}

	migration, err := launchercalc.getLatestVmimIfExist(vmi, pod.Namespace)
	if err != nil {
		log.Log.Infof("can't detect pod as launcher pod pod.name: %v\\n", pod.Name)
		return corev1.ResourceList{}, err, true
	}

	sourcePod := getSourcePod(launcherPods, vmi)
	targetPod := getTargetPod(launcherPods, migration)

	if sourcePod == nil { //check only source because target might be gated
		var launcherPodsForErr []string
		for _, pod := range launcherPods {
			launcherPodsForErr = append(launcherPodsForErr, pod.Name)
		}
		errMsg := fmt.Sprintf("something is wrong could not detect source pod launcherPods in ns: %v  source == nil: %v ", launcherPodsForErr, sourcePod == nil)
		log.Log.Warningf(errMsg)
		return corev1.ResourceList{}, fmt.Errorf(errMsg), true
	}

	sourceResources, err := launchercalc.calculateSourceUsageByConfig(&pod, vmi)
	if err != nil {
		return corev1.ResourceList{}, err, true
	}

	if pod.Name == sourcePod.Name {
		return sourceResources, nil, true
	} else if targetPod != nil && pod.Name == targetPod.Name {
		targetResources, err := launchercalc.calculateTargetUsageByConfig(&pod, vmi)
		if err != nil {
			return corev1.ResourceList{}, err, true
		}
		return v12.SubtractWithNonNegativeResult(targetResources, sourceResources), nil, true
	}
	//shouldn't get here unless we evaluate orphan launcher pod, in that case the pod will be deleted soon
	return corev1.ResourceList{}, nil, true
}

func (launchercalc *VirtLauncherCalculator) calculateSourceUsageByConfig(pod *corev1.Pod, vmi *v15.VirtualMachineInstance) (corev1.ResourceList, error) {
	return launchercalc.calculateUsageByConfig(pod, vmi, true)
}

func (launchercalc *VirtLauncherCalculator) calculateTargetUsageByConfig(pod *corev1.Pod, vmi *v15.VirtualMachineInstance) (corev1.ResourceList, error) {
	return launchercalc.calculateUsageByConfig(pod, vmi, false)
}

func (launchercalc *VirtLauncherCalculator) calculateUsageByConfig(pod *corev1.Pod, vmi *v15.VirtualMachineInstance, isSourceOrSingleLauncher bool) (corev1.ResourceList, error) {
	config := launchercalc.calcConfig
	if !validConfig(config) {
		config = DefaultLauncherConfig
	}
	switch config {
	case VmiPodUsage:
		podEvaluator := core.NewPodEvaluator(nil, clock.RealClock{})
		return podEvaluator.Usage(pod)
	case VirtualResources:
		return calculateResourceLauncherVMIUsagePodResources(vmi, isSourceOrSingleLauncher), nil
	case DedicatedVirtualResources:
		vmiRl := corev1.ResourceList{
			ResourcePodsOfVmi: *(resource.NewQuantity(1, resource.DecimalSI)),
		}
		usageToConvertVmiResources := calculateResourceLauncherVMIUsagePodResources(vmi, isSourceOrSingleLauncher)
		if memRq, ok := usageToConvertVmiResources[corev1.ResourceRequestsMemory]; ok {
			vmiRl[ResourceRequestsVmiMemory] = memRq
		}
		if memLim, ok := usageToConvertVmiResources[corev1.ResourceLimitsMemory]; ok {
			vmiRl[ResourceRequestsVmiMemory] = memLim
		}
		if cpuRq, ok := usageToConvertVmiResources[corev1.ResourceRequestsCPU]; ok {
			vmiRl[ResourceRequestsVmiCPU] = cpuRq
		}
		if cpuLim, ok := usageToConvertVmiResources[corev1.ResourceLimitsCPU]; ok {
			vmiRl[ResourceRequestsVmiCPU] = cpuLim
		}
		return vmiRl, nil
	}

	podEvaluator := core.NewPodEvaluator(nil, clock.RealClock{})
	return podEvaluator.Usage(pod)
}

func validConfig(target VmiCalcConfigName) bool {
	for _, item := range MyConfigs {
		if item == target {
			return true
		}
	}
	return false
}

func calculateResourceLauncherVMIUsagePodResources(vmi *v15.VirtualMachineInstance, isSourceOrSingleLauncher bool) corev1.ResourceList {
	result := corev1.ResourceList{
		corev1.ResourcePods: *(resource.NewQuantity(1, resource.DecimalSI)),
	}
	//todo: once memory hot-plug is implemented we need to update this
	memoryGuest := corev1.ResourceList{}
	memoryGuestHugePages := corev1.ResourceList{}
	domainResourcesReq := corev1.ResourceList{}
	domainResourcesLim := corev1.ResourceList{}

	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
		memoryGuest = corev1.ResourceList{corev1.ResourceRequestsMemory: *vmi.Spec.Domain.Memory.Guest}
	}
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		quantity, err := resource.ParseQuantity(vmi.Spec.Domain.Memory.Hugepages.PageSize)
		if err == nil {
			memoryGuestHugePages = corev1.ResourceList{corev1.ResourceRequestsMemory: quantity}
		}
	}
	if vmi.Spec.Domain.Resources.Requests != nil {
		resourceMemory, resourceMemoryExist := vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]
		resourceRequestsMemory, resourceRequestsMemoryExist := vmi.Spec.Domain.Resources.Requests[corev1.ResourceRequestsMemory]
		if resourceMemoryExist && !resourceRequestsMemoryExist {
			domainResourcesReq = corev1.ResourceList{corev1.ResourceRequestsMemory: resourceMemory}
		} else if resourceRequestsMemoryExist {
			domainResourcesReq = corev1.ResourceList{corev1.ResourceRequestsMemory: resourceRequestsMemory}
		}
	}
	if vmi.Spec.Domain.Resources.Limits != nil {
		resourceMemory, resourceMemoryExist := vmi.Spec.Domain.Resources.Limits[corev1.ResourceMemory]
		resourceLimitsMemory, resourceLimitsMemoryExist := vmi.Spec.Domain.Resources.Limits[corev1.ResourceLimitsMemory]
		if resourceMemoryExist && !resourceLimitsMemoryExist {
			domainResourcesLim = corev1.ResourceList{corev1.ResourceLimitsMemory: resourceMemory}
		} else if resourceLimitsMemoryExist {
			domainResourcesLim = corev1.ResourceList{corev1.ResourceLimitsMemory: resourceLimitsMemory}
		}
	}

	tmpMemReq := v12.Max(memoryGuest, domainResourcesReq)
	tmpMemLim := v12.Max(memoryGuest, domainResourcesLim)

	finalMemReq := v12.Max(tmpMemReq, memoryGuestHugePages)
	finalMemLim := v12.Max(tmpMemLim, corev1.ResourceList{corev1.ResourceLimitsMemory: finalMemReq[corev1.ResourceRequestsMemory]})
	requests := corev1.ResourceList{
		corev1.ResourceRequestsMemory: finalMemReq[corev1.ResourceRequestsMemory],
	}
	limits := corev1.ResourceList{
		corev1.ResourceLimitsMemory: finalMemLim[corev1.ResourceLimitsMemory],
	}

	var cpuTopology *v15.CPUTopology
	if isSourceOrSingleLauncher && vmi.Status.CurrentCPUTopology != nil {
		cpuTopology = vmi.Status.CurrentCPUTopology
	} else {
		cpuTopology = &v15.CPUTopology{
			Cores:   vmi.Spec.Domain.CPU.Cores,
			Sockets: vmi.Spec.Domain.CPU.Sockets,
			Threads: vmi.Spec.Domain.CPU.Threads,
		}
	}
	vcpus := getNumberOfVCPUs(cpuTopology)
	vcpuQuantity := *resource.NewQuantity(vcpus, resource.BinarySI)

	guestCpuReq := v12.Max(corev1.ResourceList{corev1.ResourceRequestsCPU: vcpuQuantity}, domainResourcesReq)
	guestCpuLim := v12.Max(corev1.ResourceList{corev1.ResourceLimitsCPU: vcpuQuantity}, domainResourcesLim)

	requests[corev1.ResourceRequestsCPU] = guestCpuReq[corev1.ResourceRequestsCPU]
	limits[corev1.ResourceLimitsCPU] = guestCpuLim[corev1.ResourceLimitsCPU]

	result = v12.Add(result, requests)
	result = v12.Add(result, limits)
	return result
}

func getNumberOfVCPUs(cpuSpec *v15.CPUTopology) int64 {
	vCPUs := cpuSpec.Cores
	if cpuSpec.Sockets != 0 {
		if vCPUs == 0 {
			vCPUs = cpuSpec.Sockets
		} else {
			vCPUs *= cpuSpec.Sockets
		}
	}
	if cpuSpec.Threads != 0 {
		if vCPUs == 0 {
			vCPUs = cpuSpec.Threads
		} else {
			vCPUs *= cpuSpec.Threads
		}
	}
	return int64(vCPUs)
}

func unfinishedVMIPods(aaqCli client.AAQClient, pods []corev1.Pod, vmi *v15.VirtualMachineInstance) (podsToReturn []*corev1.Pod) {
	if pods == nil {
		podsList, err := aaqCli.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			klog.Errorf("AaqGateController: Error: %v", err)
		}
		pods = podsList.Items
	}

	for _, pod := range pods {
		if !core.QuotaV1Pod(&pod, clock.RealClock{}) {
			continue
		}

		if app, ok := pod.Labels[v15.AppLabel]; !ok || app != launcherLabel {
			continue
		}
		if pod.OwnerReferences == nil {
			continue
		}

		ownerRefs := pod.GetOwnerReferences()
		found := false
		for _, ownerRef := range ownerRefs {
			if ownerRef.UID == vmi.GetUID() {
				found = true
			}
		}

		if !found {
			continue
		}
		podsToReturn = append(podsToReturn, pod.DeepCopy())
	}
	return podsToReturn
}

func podExists(pods []*corev1.Pod, targetPod *corev1.Pod) bool {
	for _, pod := range pods {
		if pod.Namespace == targetPod.Namespace &&
			pod.Name == targetPod.Name {
			return true
		}
	}
	return false
}

func (launchercalc *VirtLauncherCalculator) getLatestVmimIfExist(vmi *v15.VirtualMachineInstance, ns string) (*v15.VirtualMachineInstanceMigration, error) {
	vmimList, err := launchercalc.aaqCli.KubevirtClient().KubevirtV1().VirtualMachineInstanceMigrations(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Log.Infof("couldn't list VirtualMachineInstanceMigrations in ns:%v err:%v\n", ns, err.Error())
		return nil, fmt.Errorf("can't fetch migrations")
	}
	var latestVmim v15.VirtualMachineInstanceMigration
	latestTimestamp := metav1.Time{}

	for _, vmim := range vmimList.Items {
		if vmim.Status.Phase != v15.MigrationFailed && vmim.Spec.VMIName == vmi.Name {
			if vmim.CreationTimestamp.After(latestTimestamp.Time) {
				latestTimestamp = vmim.CreationTimestamp
				latestVmim = vmim
			}
		}
	}

	return &latestVmim, nil
}

func getSourcePod(pods []*corev1.Pod, vmi *v15.VirtualMachineInstance) *corev1.Pod {
	var curPod *corev1.Pod = nil
	for _, pod := range pods {
		if vmi.Status.NodeName != "" &&
			vmi.Status.NodeName != pod.Spec.NodeName {
			// This pod isn't scheduled to the current node.
			// This can occur during the initial migration phases when
			// a new target node is being prepared for the VMI.
			continue
		}

		if curPod == nil || curPod.CreationTimestamp.Before(&pod.CreationTimestamp) {
			curPod = pod
		}
	}

	return curPod
}

func getTargetPod(allPods []*corev1.Pod, migration *v15.VirtualMachineInstanceMigration) *corev1.Pod {
	for _, pod := range allPods {
		migrationUID, migrationLabelExist := pod.Labels[v15.MigrationJobLabel]
		migrationName, migrationAnnExist := pod.Annotations[v15.MigrationJobNameAnnotation]
		if migration != nil && migrationLabelExist && migrationAnnExist && migrationUID == string(migration.UID) && migrationName == migration.Name {
			return pod
		}
	}
	return nil
}

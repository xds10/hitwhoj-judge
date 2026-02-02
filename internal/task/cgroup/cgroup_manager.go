package cgroup

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ResourceUsage 表示资源使用情况
type ResourceUsage struct {
	CPUUsec int64 // CPU使用时间（微秒）
	MemPeak int64 // 内存峰值（字节）
}

// CgroupManager cgroup管理器
type CgroupManager struct {
	cgroupPath string
}

func detectSelfCgroup() string {
	data, _ := os.ReadFile("/proc/self/cgroup")
	// cgroup v2: 0::/system.slice/xxx.scope
	parts := strings.Split(strings.TrimSpace(string(data)), ":")
	return filepath.Join("/sys/fs/cgroup", parts[2])
}
func enableControllers(cgroupPath string) error {
	ctrl := filepath.Join(cgroupPath, "cgroup.subtree_control")
	return os.WriteFile(ctrl, []byte("+cpu +memory"), 0644)
}

// NewCgroupManager 创建新的cgroup管理器
func NewCgroupManager(id string) (*CgroupManager, error) {
	// path := filepath.Join(detectSelfCgroup(), "judge")
	// enableControllers(path)
	// path = filepath.Join(path, id)

	// if err := os.MkdirAll(path, 0777); err != nil {
	// 	return nil, fmt.Errorf("failed to create cgroup: %w", err)
	// }

	return &CgroupManager{
		cgroupPath: "path",
	}, nil
}

// SetLimits 设置资源限制
func (cm *CgroupManager) SetLimits(cpuLimitUS int64, memLimitBytes int64) error {
	// 设置CPU限制 (quota period = 100000us)
	cpuMax := fmt.Sprintf("%d 100000", cpuLimitUS)
	if err := ioutil.WriteFile(
		filepath.Join(cm.cgroupPath, "cpu.max"),
		[]byte(cpuMax),
		0644,
	); err != nil {
		return fmt.Errorf("failed to set cpu limit: %w", err)
	}

	// 设置内存限制
	if err := ioutil.WriteFile(
		filepath.Join(cm.cgroupPath, "memory.max"),
		[]byte(strconv.FormatInt(memLimitBytes, 10)),
		0644,
	); err != nil {
		return fmt.Errorf("failed to set memory limit: %w", err)
	}

	// 启用内存压力事件
	if err := ioutil.WriteFile(
		filepath.Join(cm.cgroupPath, "memory.pressure"),
		[]byte("low"),
		0644,
	); err != nil {
		// 这个可能失败，但不影响核心功能
	}

	return nil
}

// AddProcessToCgroup 将进程添加到cgroup
func (cm *CgroupManager) AddProcessToCgroup(pid int) error {
	return ioutil.WriteFile(
		filepath.Join(cm.cgroupPath, "cgroup.procs"),
		[]byte(strconv.Itoa(pid)),
		0644,
	)
}

// ReadUsage 读取资源使用情况
func (cm *CgroupManager) ReadUsage() (*ResourceUsage, error) {
	cpuStat, err := ioutil.ReadFile(filepath.Join(cm.cgroupPath, "cpu.stat"))
	if err != nil {
		return nil, fmt.Errorf("failed to read cpu.stat: %w", err)
	}

	memPeakData, err := ioutil.ReadFile(filepath.Join(cm.cgroupPath, "memory.peak"))
	if err != nil {
		// 如果没有peak文件，尝试读取current
		memPeakData, err = ioutil.ReadFile(filepath.Join(cm.cgroupPath, "memory.current"))
		if err != nil {
			return nil, fmt.Errorf("failed to read memory usage: %w", err)
		}
	}

	var cpuUsec int64
	for _, line := range strings.Split(string(cpuStat), "\n") {
		if strings.HasPrefix(line, "usage_usec ") {
			fmt.Sscanf(line, "usage_usec %d", &cpuUsec)
		}
	}

	memPeak, err := strconv.ParseInt(strings.TrimSpace(string(memPeakData)), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory peak: %w", err)
	}

	return &ResourceUsage{
		CPUUsec: cpuUsec,
		MemPeak: memPeak,
	}, nil
}

// GetCgroupPath 返回cgroup路径
func (cm *CgroupManager) GetCgroupPath() string {
	return cm.cgroupPath
}

// Cleanup 清理cgroup
func (cm *CgroupManager) Cleanup() error {
	return os.RemoveAll(cm.cgroupPath)
}

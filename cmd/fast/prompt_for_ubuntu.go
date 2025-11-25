package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/linkease/fastpve/downloader"
	"github.com/linkease/fastpve/quickget"
	"github.com/linkease/fastpve/utils"
	"github.com/manifoldco/promptui"
)

const (
	Ubuntu2204ForDesk = iota
	Ubuntu2204ForServer
	Ubuntu2410ForDesk
	Ubuntu2410ForServer
	Ubuntu2504ForDesk
	Ubuntu2504ForServer
)

type ubuntuInstallInfo struct {
	UbuntuISO    string `json:"ubuntuISO"`
	UbuntuVer    int    `json:"ubuntuVer"`
	Memory       int    `json:"memory"`
	Cores        int    `json:"cores"`
	Disk         int    `json:"disk"`
	DownloadOnly bool   `json:"downloadOnly"`
}

func promptForUbuntu() error {
	isoPath := "/var/lib/vz/template/iso/"
	cachePath := "/var/lib/vz/template/cache"
	downer := newDownloader()
	statusPath := filepath.Join(cachePath, "ubuntu_install.ops")
	status, _ := isStatusValid(downer, statusPath)

	var ubuntuISOs []string
	dirs, err := os.ReadDir(isoPath)
	if err == nil {
		ubuntuISOs = getUbuntuISO(dirs)
	}

	info := &ubuntuInstallInfo{
		UbuntuVer: -1,
	}

	err = promptUbuntuFiles(info, status, ubuntuISOs)
	if err != nil {
		return err
	}
	info.Cores, err = promptPVECore()
	if err != nil {
		return err
	}
	info.Memory, err = promptPVEMemory()
	if err != nil {
		return err
	}
	info.Disk, err = promptPVEDisk()
	if err != nil {
		return err
	}

	fmt.Println("install=", utils.ToString(info))
	var needDownload bool
	if (status != nil && info.UbuntuISO == status.TargetFile) ||
		info.UbuntuVer >= 0 {
		needDownload = true
	}
	next, err := promptUbuntuDownloadInstall(info, needDownload)
	if err != nil {
		return err
	}
	if !next {
		return nil
	}

	ctx := context.TODO()
	if status != nil && info.UbuntuISO == status.TargetFile {
		// Continue download target file
		info.UbuntuISO, err = downloadUbuntuISO(ctx, downer, status, isoPath, cachePath, statusPath)
		if err != nil {
			return err
		}
	}
	if info.UbuntuVer >= 0 {
		urls, err := getUbuntuUrls(ctx, downer, info.UbuntuVer)
		if err != nil {
			urls = defaultUbuntuUrls(info.UbuntuVer)
		}
		var urlStr string
		var totalSize int64
		var modTime time.Time
		for _, s := range urls {
			totalSize, modTime, err = downer.HeadInfo(s)
			if err == nil {
				urlStr = s
				break
			}
		}
		status = &downloader.DownloadStatus{
			Url:        urlStr,
			TargetFile: filepath.Join(cachePath, path.Base(urlStr)),
			TotalSize:  totalSize,
			ModTime:    modTime,
		}
		info.UbuntuISO, err = downloadUbuntuISO(ctx, downer, status, isoPath, cachePath, statusPath)
		if err != nil {
			return err
		}
	}
	if info.DownloadOnly {
		return nil
	}

	return createUbuntuVM(ctx, isoPath, info)
}

func promptUbuntuFiles(info *ubuntuInstallInfo, status *downloader.DownloadStatus, ubuntuISOs []string) error {
	origUbuntuLen := len(ubuntuISOs)
	if status != nil {
		name := filepath.Base(status.TargetFile)
		name = strings.TrimSuffix(name, ".syn")
		progress := status.Curr * 100 / (status.TotalSize + 1)
		name = fmt.Sprintf("继续下载 %s(%02d%%)", name, progress)
		ubuntuISOs = append(ubuntuISOs, name)
	}
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 22.04-desktop")
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 22.04-live-server")
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 24.10-desktop")
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 24.10-live-server")
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 25.04-desktop")
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 25.04-live-server")
	prompt := promptui.Select{
		Label: "选择Ubuntu安装文件",
		Items: ubuntuISOs,
	}
	idx, file, err := prompt.Run()
	if err != nil {
		return err
	}
	if idx < origUbuntuLen {
		info.UbuntuISO = file
	} else {
		if status != nil && idx == (len(ubuntuISOs)-7) {
			info.UbuntuISO = status.TargetFile
		} else if idx >= (len(ubuntuISOs) - 6) {
			info.UbuntuVer = idx - (len(ubuntuISOs) - 6)
		}
	}

	return nil
}

func getUbuntuISO(dirs []os.DirEntry) []string {
	var isoFiles []string
	for _, dir := range dirs {
		if !dir.IsDir() &&
			strings.HasPrefix(dir.Name(), "ubuntu-") &&
			filepath.Ext(dir.Name()) == ".iso" {
			isoFiles = append(isoFiles, dir.Name())
		}
	}
	return isoFiles
}

func promptUbuntuDownloadInstall(info *ubuntuInstallInfo, needDownload bool) (bool, error) {
	var items []string
	if needDownload {
		items = []string{"下载并安装", "仅下载", "退出"}
	} else {
		items = []string{"安装", "退出"}
	}
	prompt := promptui.Select{
		Label: fmt.Sprintf("选择完成，继续安装%s：（CPU：%d,内存：%dMB,硬盘：%dGB）",
			filepath.Base(info.UbuntuISO),
			info.Cores,
			info.Memory,
			info.Disk),
		Items: items,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return false, err
	}
	if idx == 0 {
		return true, nil
	}
	if needDownload {
		if idx == 1 {
			info.DownloadOnly = true
			return true, nil
		}
	}
	return false, nil
}

func defaultUbuntuUrls(ver int) []string {
	var versionStr1, versionStr2, versionStr3 string
	if ver&1 == 0 {
		versionStr3 = "desktop"
	} else {
		versionStr3 = "live-server"
	}
	switch ver {
	case Ubuntu2204ForDesk, Ubuntu2204ForServer:
		versionStr1 = "22.04"
		versionStr2 = "22.04.5"
	case Ubuntu2410ForDesk, Ubuntu2410ForServer:
		versionStr1 = "24.10"
		versionStr2 = "24.10"
	case Ubuntu2504ForDesk, Ubuntu2504ForServer:
		versionStr1 = "25.04"
		versionStr2 = "25.04"
	}
	return []string{
		fmt.Sprintf("https://mirrors.ustc.edu.cn/ubuntu-releases/%s/ubuntu-%s-%s-amd64.iso", versionStr1, versionStr2, versionStr3),
	}
}

func getUbuntuUrls(ctx context.Context, downer *downloader.Downloader, ver int) ([]string, error) {
	var versionStr1, versionStr2, versionStr3 string

	if ver&1 == 0 {
		versionStr3 = "desktop"
	} else {
		versionStr3 = "live-server"
	}

	switch ver {
	case Ubuntu2204ForDesk, Ubuntu2204ForServer:
		versionStr1 = "22.04"
		versionStr2 = "22.04.5"
	case Ubuntu2410ForDesk, Ubuntu2410ForServer:
		versionStr1 = "24.10"
		versionStr2 = "24.10"
	case Ubuntu2504ForDesk, Ubuntu2504ForServer:
		versionStr1 = "25.04"
		versionStr2 = "25.04"
	default:
		return nil, fmt.Errorf("unknown Ubuntu version: %d", ver)
	}
	return []string{
		// 中科大
		fmt.Sprintf("https://mirrors.ustc.edu.cn/ubuntu-releases/%s/ubuntu-%s-%s-amd64.iso", versionStr1, versionStr2, versionStr3),
		// 清华
		fmt.Sprintf("https://mirrors.tuna.tsinghua.edu.cn/ubuntu-releases/%s/ubuntu-%s-%s-amd64.iso", versionStr1, versionStr2, versionStr3),
		// 华为
		fmt.Sprintf("https://repo.huaweicloud.com/ubuntu-releases/%s/ubuntu-%s-%s-amd64.iso", versionStr1, versionStr2, versionStr3),
		// 官方
		fmt.Sprintf("https://releases.ubuntu.com/%s/ubuntu-%s-%s-amd64.iso", versionStr1, versionStr2, versionStr3),
	}, nil
}

func downloadUbuntuISO(ctx context.Context,
	downer *downloader.Downloader,
	status *downloader.DownloadStatus,
	isoPath, cachePath, statusPath string) (string, error) {
	baseFileName := filepath.Base(status.TargetFile)
	fmt.Println("downloading:", baseFileName, "url=\n", status.Url)
	err := downloadURL(ctx, downer, statusPath, status)
	if err != nil {
		return "", err
	}
	targetFilePath := filepath.Join(isoPath, baseFileName)
	err = os.Rename(status.TargetFile, targetFilePath)
	if err != nil {
		return "", err
	}
	return targetFilePath, nil
}

func createUbuntuVM(ctx context.Context, isoPath string, info *ubuntuInstallInfo) error {
	disks, err := quickget.DiskStatus()
	if err != nil {
		return err
	}
	useDisk := "local"
	if len(disks) > 0 {
		useDisk = disks[0]
	}
	for _, disk := range disks {
		if disk == "local-lvm" {
			useDisk = "local-lvm"
			break
		}
	}

	items, err := quickget.QMList()
	if err != nil {
		return err
	}
	vmid := 100
	if len(items) > 0 {
		sort.Slice(items, func(i, j int) bool {
			return items[i].VMID < items[j].VMID
		})
		vmid = items[len(items)-1].VMID + 1
	}
	imgName := filepath.Base(info.UbuntuISO)
	vmName := toBetterUbuntuName(imgName)
	scripts := []string{
		"set -e",
		`export LC_ALL="en_US.UTF-8"`,
		fmt.Sprintf("export VMID=%d", vmid),
		fmt.Sprintf(`qm create $VMID --name "%s" --memory %d --scsihw virtio-scsi-single --cores %d --sockets 1 --machine q35 --bios ovmf --cpu host --net0 virtio,bridge=vmbr0`,
			vmName,
			info.Memory,
			info.Cores),
		fmt.Sprintf("qm set $VMID -efidisk0 %s:1,format=raw,efitype=4m", useDisk),
		fmt.Sprintf("qm set $VMID --scsi0 %s:%d", useDisk, info.Disk),
		fmt.Sprintf(`qm set $VMID --ide0 local:iso/%s,media=cdrom`, imgName),
		`qm set $VMID --boot order='scsi0;ide0'`,
		`qm set $VMID --agent enabled=1,fstrim_cloned_disks=1`,
		`qm set $VMID --ostype l26`,
		`echo "VMOK"`,
	}
	//fmt.Println(strings.Join(scripts, "\n"))
	out, err := utils.BatchOutput(ctx, scripts, 0)
	if err != nil {
		return err
	}
	if strings.Contains(string(out), "VMOK") {
		fmt.Println("创建虚拟机：", vmid, "成功，请到网页端启动虚拟机并继续安装系统")
		return nil
	}
	return errors.New("VM creation failed")
}

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
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

var winEditions = []string{
	"Chinese (Simplified)",
	"Chinese (Traditional)",
	"English (United States)",
	"English International",
}

const (
	Win11 = iota
	Win10
)

type windowsInstallInfo struct {
	WindowISO    string `json:"windowISO"`
	VirtIO       string `json:"virtio"`
	WinVersion   int    `json:"winVersion"` // 0:11, 1:10
	WinEdition   int    `json:"winEdition"`
	Memory       int    `json:"memory"`
	Cores        int    `json:"cores"`
	Disk         int    `json:"disk"`
	DownloadOnly bool   `json:"downloadOnly"`
}

func promptInstallWindows() error {
	isoPath := "/var/lib/vz/template/iso/"
	cachePath := "/var/lib/vz/template/cache"
	downer := newDownloader()
	statusPath := filepath.Join(cachePath, "windows_install.ops")
	status, _ := isStatusValid(downer, statusPath)

	var windows []string
	var virtio []string
	dirs, err := os.ReadDir(isoPath)
	if err == nil {
		windows = getWindowISO(dirs)
		virtio = getVirtIOISO(dirs)
	}

	info := &windowsInstallInfo{
		WinVersion: -1,
		WinEdition: -1,
	}

	err = promptWinFiles(info, status, windows)
	if err != nil {
		return err
	}
	if len(virtio) > 0 {
		prompt := promptui.Select{
			Label: "选择VirtIO驱动文件",
			Items: virtio,
		}
		var file string
		_, file, err = prompt.Run()
		if err != nil {
			return err
		}
		info.VirtIO = file
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
	if (status != nil && info.WindowISO == status.TargetFile) ||
		info.WinVersion >= 0 && info.WinEdition >= 0 ||
		info.VirtIO == "" {
		needDownload = true
	}
	next, err := promptWinDownloadInstall(info, needDownload)
	if err != nil {
		return err
	}
	if !next {
		return nil
	}

	ctx := context.TODO()
	var hasJq, uuidgen bool
	if _, err := exec.LookPath("jq"); err == nil {
		hasJq = true
	}
	if _, err := exec.LookPath("uuidgen"); err == nil {
		uuidgen = true
	}
	if !hasJq || !uuidgen {
		fmt.Println("安装缺失的 jq uuidgen")
		utils.BatchRunStdout(ctx, []string{"apt update && apt install -y jq uuid-runtime"}, 0)
		if _, err := exec.LookPath("jq"); err == nil {
			hasJq = true
		}
		if _, err := exec.LookPath("uuidgen"); err == nil {
			uuidgen = true
		}
		if !hasJq {
			fmt.Println("缺少 jq 而且再次安装也失败")
			return errors.New("缺少 jq")
		}
		if !uuidgen {
			fmt.Println("缺少 uuidgen 而且再次安装也失败")
			return errors.New("缺少 uuidgen")
		}
	}

	quickGet, err := quickget.CreateQuickGet()
	if err != nil {
		return err
	}
	defer os.Remove(quickGet)
	//log.Println("quickGet=", quickGet)

	if status != nil && info.WindowISO == status.TargetFile {
		// Continue download target file
		realPath := strings.TrimSuffix(status.TargetFile, ".syn")
		fmt.Println("downloading:", filepath.Base(realPath))
		err = downloadURL(ctx, downer, statusPath, status)
		if err != nil {
			return err
		}
		info.WindowISO = realPath
		err = os.Rename(status.TargetFile, info.WindowISO)
		if err != nil {
			return err
		}
	}
	if info.WinVersion >= 0 && info.WinEdition >= 0 {
		if status != nil {
			os.Remove(status.TargetFile)
			os.Remove(statusPath)
			status = nil
		}
		// Create new download file
		var winVer string
		if info.WinVersion == Win11 {
			winVer = "11"
		} else {
			winVer = "10"
		}
		tag := strings.Join([]string{
			"windows",
			winVer,
			utils.CleanString(winEditions[info.WinEdition]),
		}, "-")
		args := []string{"--url", "windows", winVer, winEditions[info.WinEdition]}
		fmt.Println("获取下载URL...")
		var totalSize int64
		var modTime time.Time
		urlStr, _ := quickget.GetSystemURL(ctx, quickGet, args)
		if urlStr != "" {
			fmt.Println("获取下载URL成功，开始下载:", urlStr)
			if err := downer.PutRemoteURL(ctx, tag, urlStr); err != nil && !errors.Is(err, downloader.ErrRemoteURLCacheDisabled) {
				return err
			}
			totalSize, modTime, err = downer.HeadInfo(urlStr)
			if err != nil {
				return err
			}
		} else if downer.RemoteURLCacheEnabled() {
			fmt.Println("获取下载URL失败，重新获取...")
			urls, err := downer.GetRemoteURLs(ctx, tag)
			if err != nil {
				return err
			}
			for _, urlTmp := range urls {
				if strings.Contains(urlTmp, "virtio-win") {
					continue
				}
				totalSize, modTime, err = downer.HeadInfo(urlTmp)
				if err == nil {
					urlStr = urlTmp
					fmt.Println("重新获取URL成功，开始下载:", urlStr)
					break
				}
			}
		} else {
			return errors.New("获取下载URL失败，且未启用远程缓存")
		}
		status = &downloader.DownloadStatus{
			Url:        urlStr,
			TargetFile: filepath.Join(isoPath, tag+".iso.syn"),
			TotalSize:  totalSize,
			ModTime:    modTime,
		}
		realPath := strings.TrimSuffix(status.TargetFile, ".syn")
		fmt.Println("downloading:", filepath.Base(realPath))
		err = downloadURL(ctx, downer, statusPath, status)
		if err != nil {
			return err
		}
		info.WindowISO = realPath
		err = os.Rename(status.TargetFile, info.WindowISO)
		if err != nil {
			return err
		}
	}

	if info.VirtIO == "" {
		virtStatusPath := filepath.Join(cachePath, "windows_virtio.ops")
		virtStatus, _ := isStatusValid(downer, virtStatusPath)
		var realPath string
		realPath, err = downloadVirtIO(ctx, downer, isoPath, virtStatusPath, virtStatus)
		if err != nil {
			return err
		}
		info.VirtIO = realPath
		err = os.Rename(virtStatus.TargetFile, realPath)
		if err != nil {
			return err
		}
	}
	if info.DownloadOnly {
		return nil
	}

	return createWindowVM(ctx, info)
}

func getWindowISO(dirs []os.DirEntry) []string {
	var isoFiles []string
	for _, dir := range dirs {
		if !dir.IsDir() &&
			strings.HasPrefix(dir.Name(), "windows-") &&
			filepath.Ext(dir.Name()) == ".iso" {
			isoFiles = append(isoFiles, dir.Name())
		}
	}
	return isoFiles
}

func getVirtIOISO(dirs []os.DirEntry) []string {
	var isoFiles []string
	for _, dir := range dirs {
		if !dir.IsDir() &&
			strings.HasPrefix(dir.Name(), "virtio-win-") &&
			filepath.Ext(dir.Name()) == ".iso" {
			isoFiles = append(isoFiles, dir.Name())
		}
	}
	return isoFiles
}

func promptWinFiles(info *windowsInstallInfo,
	status *downloader.DownloadStatus,
	windows []string) error {
	origWinLen := len(windows)
	if status != nil {
		name := filepath.Base(status.TargetFile)
		name = strings.TrimSuffix(name, ".syn")
		progress := status.Curr * 100 / (status.TotalSize + 1)
		name = fmt.Sprintf("继续下载 %s(%02d%%)", name, progress)
		windows = append(windows, name)
	}
	windows = append(windows, "全新下载 Windows11")
	windows = append(windows, "全新下载 Windows10")
	prompt := promptui.Select{
		Label: "选择Windows安装文件",
		Items: windows,
	}
	idx, file, err := prompt.Run()
	if err != nil {
		return err
	}
	var selWin bool
	if idx < origWinLen {
		info.WindowISO = file
	} else {
		if status != nil && idx == (len(windows)-3) {
			info.WindowISO = status.TargetFile
		} else if idx >= (len(windows) - 2) {
			selWin = true
			info.WinVersion = idx - (len(windows) - 2)
			err = promptWinEdition(info)
			if err != nil {
				return err
			}
		}
	}
	if !selWin {
		prompt := promptui.Select{
			Label: "选择系统：",
			Items: []string{"11", "10"},
		}
		idx, _, err := prompt.Run()
		if err != nil {
			return err
		}
		info.WinVersion = idx
	}

	return nil
}

func promptWinEdition(info *windowsInstallInfo) error {
	prompt := promptui.Select{
		Label: "Windows版本语言",
		Items: winEditions,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return err
	}
	info.WinEdition = idx
	return nil
}

func promptWinDownloadInstall(info *windowsInstallInfo, needDownload bool) (bool, error) {
	var items []string
	if needDownload {
		items = []string{"下载并安装", "仅下载", "退出"}
	} else {
		items = []string{"安装", "退出"}
	}
	prompt := promptui.Select{
		Label: fmt.Sprintf("选择完成，继续安装%s：（CPU：%d,内存：%dMB,硬盘：%dGB）",
			filepath.Base(info.WindowISO),
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

func downloadVirtIO(ctx context.Context,
	downer *downloader.Downloader,
	isoPath string,
	virtStatusPath string,
	virtStatus *downloader.DownloadStatus) (string, error) {
	if virtStatus != nil {
		realPath := strings.TrimSuffix(virtStatus.TargetFile, ".syn")
		fmt.Println("downloading:", filepath.Base(realPath), "url=\n", virtStatus.Url)
		err := downloadURL(ctx, downer, virtStatusPath, virtStatus)
		if err == nil {
			return realPath, nil
		}
	}

	urls := []string{
		"https://dl.istoreos.com/iStoreOS/Virtual/virtio-win-0.1.271.iso",
		"https://fw0.koolcenter.com/iStoreOS/Virtual/virtio-win-0.1.271.iso",
		"https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.271-1/virtio-win-0.1.271.iso",
	}
	var virtioURL string
	var totalSize int64
	var modTime time.Time
	var err error
	for _, s := range urls {
		totalSize, modTime, err = downer.HeadInfo(s)
		if err == nil {
			virtioURL = s
			break
		}
	}

	virtStatus = &downloader.DownloadStatus{
		Url:        virtioURL,
		TargetFile: filepath.Join(isoPath, path.Base(virtioURL)+".syn"),
		TotalSize:  totalSize,
		ModTime:    modTime,
	}

	realPath := strings.TrimSuffix(virtStatus.TargetFile, ".syn")
	fmt.Println("downloading:", filepath.Base(realPath), "url=\n", virtStatus.Url)
	err = downloadURL(ctx, downer, virtStatusPath, virtStatus)
	return realPath, err
}

func createWindowVM(ctx context.Context, info *windowsInstallInfo) error {
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
	winID := 10
	var tpmStr string
	if info.WinVersion == Win11 {
		winID = 11
		tpmStr = fmt.Sprintf(`qm set $VMID -tpmstate0 %s:1,version=v2.0`, useDisk)
	} else {
		tpmStr = `echo win10`
	}
	winName := filepath.Base(info.WindowISO)
	vmName := toBetterWindowName(winName)
	scripts := []string{
		"set -e",
		`export LC_ALL="en_US.UTF-8"`,
		fmt.Sprintf("export VMID=%d", vmid),
		fmt.Sprintf(`qm create $VMID --name "%s" --memory %d --scsihw virtio-scsi-single --cores %d --sockets 1 --machine q35 --bios ovmf --cpu host --net0 virtio,bridge=vmbr0`,
			vmName,
			info.Memory,
			info.Cores),
		fmt.Sprintf("qm set $VMID -efidisk0 %s:1,format=raw,efitype=4m,pre-enrolled-keys=1", useDisk),
		fmt.Sprintf("qm set $VMID --scsi0 %s:64", useDisk),
		fmt.Sprintf(`qm set $VMID --ide0 local:iso/%s,media=cdrom`, winName),
		fmt.Sprintf(`qm set $VMID --ide1 local:iso/%s,media=cdrom`, filepath.Base(info.VirtIO)),
		`qm set $VMID --boot order='scsi0;ide0;ide1'`,
		`qm set $VMID --agent enabled=1,fstrim_cloned_disks=1`,
		tpmStr,
		fmt.Sprintf("qm set %d --ostype win%d", vmid, winID),
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

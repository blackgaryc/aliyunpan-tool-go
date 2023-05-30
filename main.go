package main

import (
	"bufio"
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/library-go/jsonhelper"
)

type (
	userpw struct {
		UserName     string                  `json:"username"`
		Password     string                  `json:"password"`
		RefreshToken string                  `json:"refreshToken"`
		WebToken     aliyunpan.WebLoginToken `json:"webToken"`
	}
)

var totalFile int64 = 0

func main() {
	panClient, ui := doLogin()
	finfo, errFinfo := panClient.FileInfoByPath(ui.FileDriveId, os.Args[1])
	if errFinfo != nil {
		log.Fatalln(errFinfo)
		fmt.Println(finfo, errFinfo)
	}
	dataFile, _ := os.OpenFile("data_"+finfo.FileName+".json", os.O_CREATE|os.O_RDWR, 0644)
	//process data last requested
	fileScanner := bufio.NewScanner(dataFile)
	fileScanner.Split(bufio.ScanLines)
	filelist := list.New().Init()
	for fileScanner.Scan() {
		line := fileScanner.Text()
		file := &aliyunpan.FileEntity{}
		err := json.Unmarshal([]byte(line), file)
		if err != nil {
			fmt.Println(err)
		}
		filelist.PushBack(*file)
	}

	listparam := &aliyunpan.FileListParam{}
	listparam.DriveId = ui.FileDriveId
	listparam.ParentFileId = finfo.FileId
	fl, errListGetAll := panClient.FileListGetAll(listparam, 1000)
	if nil != errListGetAll {
		fmt.Println(fl, errListGetAll)
		return
	}

	numFile, numDir := fl.Count()
	totalFile += numFile
	fmt.Fprintln(os.Stdout, "totalFile:\t\t", totalFile)
	for i := 0; i < int(numDir+numFile); i++ {
		file := fl.Item(i)
		fmt.Println(file.FileName)
		jsonhelper.MarshalData(dataFile, file)
		if file.FileType == "folder" && !isNotLastFolder(filelist, file) {
			time.Sleep(1 * time.Second)
			ListSubDir(panClient, ui, file, dataFile)
		}
	}
}
func ListSubDir(panclient *aliyunpan.PanClient, userinfo *aliyunpan.UserInfo, pfile *aliyunpan.FileEntity, datafile *os.File) {
	listparam := &aliyunpan.FileListParam{}
	listparam.DriveId = userinfo.FileDriveId
	listparam.ParentFileId = pfile.FileId
	fl, errListGetAll := panclient.FileListGetAll(listparam, 1200)
	if nil != errListGetAll {
		fmt.Println(fl, errListGetAll)
		panclient, userinfo = doLogin()
		fl, _ = panclient.FileListGetAll(listparam, 1200)
	}
	numFile, numDir := fl.Count()
	totalFile += numFile
	fmt.Fprintln(os.Stdout, "totalFile:\t\t", totalFile)
	for i := 0; i < int(numDir+numFile); i++ {
		file := fl.Item(i)
		fmt.Println(file.FileName)
		jsonhelper.MarshalData(datafile, file)
		if file.FileType == "folder" {
			time.Sleep(1 * time.Second)
			ListSubDir(panclient, userinfo, file, datafile)
			// mvSubFolderItems2Parent(panclient, userinfo, file, pfile)
		}
	}
}
func mvSubFolderItems2Parent(panclient *aliyunpan.PanClient, userinfo *aliyunpan.UserInfo, pfile *aliyunpan.FileEntity, ppfile *aliyunpan.FileEntity) {
	// mv := &aliyunpan.FileMoveParam{}
	// panclient.FileMove(mv)
	listparam := &aliyunpan.FileListParam{}
	listparam.DriveId = userinfo.FileDriveId
	listparam.ParentFileId = pfile.FileId
	fl, errListGetAll := panclient.FileListGetAll(listparam, 1200)
	if nil != errListGetAll {
		fmt.Println(fl, errListGetAll)
		panclient, userinfo = doLogin()
	}
	numFile, numDir := fl.Count()
	totalFile += numFile
	fmt.Fprintln(os.Stdout, "totalFile:\t\t", totalFile)
	var filearr [](*aliyunpan.FileMoveParam)
	for i := 0; i < int(numDir+numFile); i++ {
		file := fl.Item(i)
		fmt.Print(file.FileName)
		tmpMv := &aliyunpan.FileMoveParam{}
		tmpMv.DriveId = userinfo.FileDriveId
		tmpMv.FileId = file.FileId
		tmpMv.ToDriveId = userinfo.FileDriveId
		tmpMv.ToParentFileId = pfile.ParentFileId
		filearr = append(filearr, tmpMv)
		fmt.Printf("filearr: %v\n", filearr)
		time.Sleep(1 * time.Second)
		panclient.FileMove(filearr)
	}
}
func doLogin() (*aliyunpan.PanClient, *aliyunpan.UserInfo) {
	configFile, err := os.OpenFile("config.json", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fmt.Println("read user info error")
		// return nil, nil
	}
	defer configFile.Close()

	localconfig := &userpw{}
	err = jsonhelper.UnmarshalData(configFile, localconfig)
	if err != nil {
		fmt.Println("read user info error")
		return nil, nil
	}
	fmt.Println(localconfig.WebToken)
	webToken := &aliyunpan.WebLoginToken{}
	webToken = &localconfig.WebToken
	if localconfig.WebToken.AccessToken == "" {
		tmpWebToken, err := aliyunpan.GetAccessTokenFromRefreshToken(localconfig.RefreshToken)
		if err != nil {
			fmt.Println("get acccess token error")
			return nil, nil
		}
		localconfig.WebToken = *tmpWebToken
		// jsonhelper.MarshalData(configFile, localconfig)
	}

	appConfig := aliyunpan.AppConfig{
		AppId:     "25dzX3vbYqktVxyX",
		DeviceId:  "WnIrdZj0pnOZpaR0LOnrsI5O",
		UserId:    "b88f5ca8f4d2454b80e1e3c419b95f7d",
		Nonce:     0,
		PublicKey: "",
	}
	// pan client
	panClient := aliyunpan.NewPanClient(*webToken, aliyunpan.AppLoginToken{}, appConfig, aliyunpan.SessionConfig{
		DeviceName: "Chrome浏览器",
		ModelName:  "Windows网页版",
	})
	r, e := panClient.CreateSession(nil)
	if e != nil {
		fmt.Println("call CreateSession error in SetupUserByCookie: " + e.Error())
	}
	if r != nil && !r.Result {
		fmt.Println("上传签名秘钥失败，可能是你账号登录的设备已超最大数量")
	}
	// get user info
	ui, errUserInfo := panClient.GetUserInfo()
	if nil != errUserInfo {
		fmt.Println("get user info error")
		fmt.Println(ui, errUserInfo)
		return nil, nil
	}
	return panClient, ui
}

func isNotLastFolder(li *list.List, file *aliyunpan.FileEntity) bool {

	var lastFolder aliyunpan.FileEntity
	for e := li.Back(); e != nil; e = e.Prev() {
		item := aliyunpan.FileEntity(e.Value.(aliyunpan.FileEntity))
		if item.FileType == "folder" {
			lastFolder = item
			break
		}
	}
	if lastFolder.FileId == file.FileId {
		return true
	}
	for {
		parentFloder, msg := getParentFile(li, &lastFolder)
		if msg != "" {
			break
		}
		if parentFloder.FileId == file.FileId {
			return true
		}
		lastFolder = *parentFloder
	}
	return false
}

func getParentFile(li *list.List, file *aliyunpan.FileEntity) (*aliyunpan.FileEntity, string) {
	var lastFolder aliyunpan.FileEntity
	for e := li.Back(); e != nil; e = e.Prev() {
		item := aliyunpan.FileEntity(e.Value.(aliyunpan.FileEntity))
		if item.FileType == "folder" && item.FileId == file.ParentFileId {
			lastFolder = item
			fmt.Println(lastFolder)
			return &lastFolder, ""
		}
	}
	return nil, "no more data"
}

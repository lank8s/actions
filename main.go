package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

var (
	client   docker.Client
	server   string
	username string
	password string
)

func init() {
	tmpClient, err := docker.NewClientFromEnv()
	if err != nil {
		panic(err)
	}

	client = *tmpClient
}

type SyncResp struct {
	Tasks []SyncImageTask `json:"tasks"`
}

type SyncImageTask struct {
	OrgRepoName    string `json:"orgRepoName"`
	TargetRepoName string `json:"targetRepoName"`
	Tag            string `json:"tag"`
}

func main() {

	flag.StringVar(&server, "s", "", "服务器地址")
	flag.StringVar(&username, "u", "username", "账号，默认为username")
	flag.StringVar(&password, "p", "password", "密码，默认为password")
	flag.Parse()
	if server == "" {
		log.Println("please input server url")
		return
	}

	resp, err := http.Get(server)
	if err != nil {
		fmt.Printf("http.Get()函数执行错误,错误为:%v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("ioutil.ReadAll()函数执行出错,错误为:%v\n", err)
		return
	}

	log.Println(string(body))

	syncResp := &SyncResp{}

	if err := json.Unmarshal(body, &syncResp); err == nil {
		log.Println("struct people:")
		log.Println(syncResp)
	}

	// if 2 > 1 {
	// 	os.Exit(-1)
	// }

	taskSize := len(syncResp.Tasks)
	if taskSize > 0 {
		var wg sync.WaitGroup
		wg.Add(taskSize)
		for i := 0; i < taskSize; i++ {
			task := syncResp.Tasks[i]
			go syncImage(task.OrgRepoName, task.TargetRepoName, task.Tag, &wg)
		}
		log.Println("wait for image sync")
		wg.Wait()
	}

	log.Println("image sync success")

	// orgRepoName := "registry.cn-shenzhen.aliyuncs.com/lan-k8s/kube-proxy"
	// targetRepoName := "registry.cn-shenzhen.aliyuncs.com/lan-k8s/kube-proxy-demo2"
	// tag := "v1.6.3"
	// syncImage(orgRepoName, targetRepoName, tag)

}

func syncImage(orgRepoName, targetRepoName, tag string, wg *sync.WaitGroup) error {

	opts := docker.PullImageOptions{
		Repository: orgRepoName,
		Tag:        tag,
		// OutputStream: &buf,
	}
	log.Printf("begin pull image|%s|%s", orgRepoName, tag)
	err := client.PullImage(opts, docker.AuthConfiguration{})
	if err != nil {
		log.Printf("pull failed:%s", err)
		defer wg.Done()
		return err
	}

	tagOpts := docker.TagImageOptions{
		Repo: targetRepoName,
		Tag:  tag,
	}
	log.Printf("begin tag image|%s|%s| to |%s|%s|", orgRepoName, tag, targetRepoName, tag)
	err = client.TagImage(orgRepoName+":"+tag, tagOpts)
	if err != nil {
		log.Printf("tag failed:%s", err)
		defer wg.Done()
		return err
	}

	// username := os.Getenv("username")
	// password := os.Getenv("password")
	//tb92722137
	//lanren123
	// log.Printf("username and password:%s|%s", username, password)
	opts2 := docker.PushImageOptions{
		Name: targetRepoName,
		Tag:  tag,
	}
	log.Printf("begin push image|%s|%s| ", targetRepoName, tag)
	err = client.PushImage(opts2, docker.AuthConfiguration{
		Username: username,
		Password: password,
	})

	if err != nil {
		log.Printf("push failed:%s", err)
		defer wg.Done()
		return err
	}
	strs := strings.Split(targetRepoName, "/")
	accessUrl := "lank8s.cn"
	for i := 1; i < len(strs); i++ {
		accessUrl += "/"
		accessUrl += strs[i]
	}
	accessUrl += ":"
	accessUrl += tag
	log.Println("you can use `docker pull " + accessUrl + "` for pull this image! \n 你可以用命令:`docker pull " + accessUrl + "`来使用这个镜像")
	defer wg.Done()
	// log.Println("sync success")
	return nil
}

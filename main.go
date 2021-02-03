package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/Albert-Zhan/httpc"
	common "github.com/DanielXLee/jd_seckill_go/common"
	conf "github.com/DanielXLee/jd_seckill_go/config"
	jd_seckill "github.com/DanielXLee/jd_seckill_go/seckill"
	"github.com/tidwall/gjson"
	"k8s.io/klog"
)

var client *httpc.HttpClient

var cookieJar *httpc.CookieJar

var config *conf.Config

var wg *sync.WaitGroup

func init() {
	//客户端设置初始化
	client = httpc.NewHttpClient()
	cookieJar = httpc.NewCookieJar()
	client.SetCookieJar(cookieJar)
	client.SetRedirect(func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	})
	//配置文件初始化
	confFile := "./conf.ini"
	if !common.Exists(confFile) {
		klog.Info("配置文件不存在，程序退出")
		os.Exit(0)
	}
	config = &conf.Config{}
	config.InitConfig(confFile)

	wg = new(sync.WaitGroup)
	wg.Add(1)
}

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()
	runtime.GOMAXPROCS(runtime.NumCPU())

	//用户登录
	user := jd_seckill.NewUser(client, config)
	wlfstkSmdl, err := user.QrLogin()
	if err != nil {
		os.Exit(0)
	}
	ticket := ""
	for {
		ticket, err = user.QrcodeTicket(wlfstkSmdl)
		if err == nil && ticket != "" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	_, err = user.TicketInfo(ticket)
	if err == nil {
		klog.Info("登录成功")
		//刷新用户状态和获取用户信息
		if status := user.RefreshStatus(); status == nil {
			userInfo, _ := user.GetUserInfo()
			klog.Infof("用户: %s", userInfo)
			//开始预约,预约过的就重复预约
			seckill := jd_seckill.NewSeckill(client, config)
			seckill.MakeReserve()
			//等待抢购/开始抢购
			nowLocalTime := time.Now().UnixNano() / 1e6
			jdTime, _ := getJdTime()
			buyDate := config.Read("config", "buy_time")
			loc, _ := time.LoadLocation("Local")
			t, _ := time.ParseInLocation("2006-01-02 15:04:05", buyDate, loc)
			buyTime := t.UnixNano() / 1e6
			diffTime := nowLocalTime - jdTime
			klog.Infof("正在等待到达设定时间: %s，检测本地时间与京东服务器时间误差为【%d】毫秒", buyDate, diffTime)
			timerTime := (buyTime + diffTime) - jdTime
			if timerTime <= 0 {
				klog.Info("请设置抢购时间")
				os.Exit(0)
			}
			time.Sleep(time.Duration(timerTime) * time.Millisecond)
			//开启任务
			klog.Info("时间到达，开始执行……")
			start(seckill, 5)
			wg.Wait()
		}
	} else {
		klog.Info("登录失败")
	}
}

func getJdTime() (int64, error) {
	req := httpc.NewRequest(client)
	resp, body, err := req.SetUrl("https://a.jd.com//ajax/queryServerData.html").SetMethod("get").Send().End()
	if err != nil || resp.StatusCode != http.StatusOK {
		klog.Info("获取京东服务器时间失败")
		return 0, fmt.Errorf("获取京东服务器时间失败")
	}
	return gjson.Get(body, "serverTime").Int(), nil
}

func start(seckill *jd_seckill.Seckill, taskNum int) {
	for i := 1; i <= taskNum; i++ {
		go func(seckill *jd_seckill.Seckill) {
			seckill.RequestSeckillUrl()
			seckill.SeckillPage()
			seckill.SubmitSeckillOrder()
		}(seckill)
	}
}

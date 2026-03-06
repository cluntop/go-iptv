package until

import (
	"context"
	"go-iptv/dao"
	"go-iptv/models"
	"log"
	"strconv"
	"sync"
	"time"
)

var Cache *SignalExecutor

type SignalExecutor struct {
	delay     time.Duration
	execFunc  func(ctx context.Context)
	signalCh  chan struct{}
	stopCh    chan struct{}
	cancel    context.CancelFunc
	timerMu   sync.Mutex
	waitTimer *time.Timer
}

// 创建 SignalExecutor 实例
func NewSignalExecutor(delay time.Duration, execFunc func(ctx context.Context)) *SignalExecutor {
	return &SignalExecutor{
		delay:    delay,
		execFunc: execFunc,
		signalCh: make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
	}
}

// 启动信号监听器
func (s *SignalExecutor) Start() {
	go func() {
		for {
			select {
			case <-s.stopCh:
				log.Println("🛑 EPG缓存重建定时任务 已停止")
				return
			case <-s.signalCh:
				s.handleSignal()
			}
		}
	}()
}

// 外部调用此函数发出信号
func (s *SignalExecutor) Rebuild() {
	select {
	case s.signalCh <- struct{}{}:
	default:
		// 若通道已满，忽略（表示已有信号等待处理）
	}
}

// 停止执行器
func (s *SignalExecutor) Stop() {
	close(s.stopCh)
	s.timerMu.Lock()
	if s.waitTimer != nil {
		s.waitTimer.Stop()
	}
	if s.cancel != nil {
		s.cancel()
	}
	s.timerMu.Unlock()
}

// 内部信号处理逻辑
func (s *SignalExecutor) handleSignal() {
	s.timerMu.Lock()
	defer s.timerMu.Unlock()

	// 如果任务正在执行 → 先中断
	if s.cancel != nil {
		log.Println("⛔ 中断当前执行EPG缓存重建任务")
		s.cancel()
		s.cancel = nil
	}

	// 若已有计时器 → 重置计时
	if s.waitTimer != nil {
		s.waitTimer.Stop()
		s.waitTimer.Reset(s.delay)
		log.Println("🔁 重置EPG缓存重建任务等待 10 秒")
		return
	}

	// 新建计时器
	log.Println("⏳ 收到EPG缓存重建任务，10 秒后执行")
	s.waitTimer = time.AfterFunc(s.delay, func() {
		s.timerMu.Lock()
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		s.waitTimer = nil
		s.timerMu.Unlock()

		log.Println("🚀 开始执行EPG缓存重建任务")
		s.execFunc(ctx)
	})
}

func doRebuild(ctx context.Context) {
	select {
	case <-ctx.Done():
		log.Println("⚠️ 重建任务被中断")
		return
	default:
		makeMealsEpgCacheAll()
		log.Println("✅ EPG缓存重建任务执行完成")
		cfg := dao.GetConfig()
		if cfg.Resolution.Auto == 1 {

			res, err := dao.WS.SendWS(dao.Request{Action: "testResolutionAll"}) //测试分辨率
			if err != nil {
				log.Println("引擎连接失败:", err)
			} else if res.Code != 1 {
				log.Println("分辨率测试失败:", res.Msg)
			} else {
				log.Println("🚀 开始执行分辨率全量识别任务，测试期间cpu、内存占用会较高，请耐心等待，强制中断执行请关闭自动测试并重启引擎")

				res, _ := dao.WS.SendWS(dao.Request{Action: "getTestStatus"}) //获取测试状态
				for res.Code != 1 {
					time.Sleep(5 * time.Second)
					res, _ = dao.WS.SendWS(dao.Request{Action: "getTestStatus"}) //获取测试状态
				}
				log.Println("分辨率测试完成")
				log.Println("🚀 重新执行EPG缓存重建")
				dao.Cache.Clear() //清除缓存
				makeMealsEpgCacheAll()
				log.Println("✅ EPG缓存重建任务执行完成")
			}
		}
	}
}

func InitCacheRebuild() {
	// 创建执行器：任务为打印模拟执行
	Cache = NewSignalExecutor(10*time.Second, doRebuild)
	log.Println("🔧 EPG缓存重建任务初始化完成")
	log.Println("入群密码前半段: 052a8103   后半段在后台>进阶功能的开发人员工具(F12)中查看")

	// 启动执行器
	Cache.Start()
	go initEpgCache()
	select {}
}

func initEpgCache() {
	log.Println("初始化订阅套餐EPG缓存")
	dao.Cache.Delete("rssEpgXml_*")
	Cache.Rebuild()
}

func CleanMealsEpgCacheAll() {
	log.Println("清理订阅套餐EPG缓存")
	dao.Cache.Delete("rssEpgXml_*")
	Cache.Rebuild()
}

func CleanAll() {
	dao.Cache.Clear()
	Cache.Rebuild()
}

func makeMealsEpgCacheAll() {
	var meals []models.IptvMeals
	dao.DB.Model(&models.IptvMeals{}).Where("status = 1").Find(&meals)
	for _, meal := range meals {
		GetEpg(meal.ID)
	}
}

func CleanMealsXmlCacheOne(id int64) {
	log.Println("删除套餐EPG订阅缓存: ", id)
	dao.Cache.Delete("rssEpgXml_" + strconv.FormatInt(id, 10))
	GetEpg(id)
}

func CleanMealsRssCacheAll() {
	dao.Cache.Delete("rssMeal*")
	dao.Cache.Delete("mytvMeal*")
}

func CleanMealsCacheAllRebuild() {
	dao.Cache.Delete("rssMeal*")
	dao.Cache.Delete("mytvMeal*")
	dao.Cache.Delete("rssEpgXml_*")
	CleanMealsEpgCacheAll()
}

func CleanMealsCacheOne(id int64) {
	log.Println("删除套餐订阅缓存: ", id)
	dao.Cache.Delete("rssMealTxt_" + strconv.FormatInt(id, 10))
	dao.Cache.Delete("rssMealM3u8_" + strconv.FormatInt(id, 10))
	dao.Cache.Delete("mytvMeal*")
}

func CleanAutoCacheAll() {
	dao.Cache.Delete("autoCategory_*")
	CleanMealsRssCacheAll()
}

func CleanAutoCacheAllRebuild() {
	dao.Cache.Delete("autoCategory_*")
	CleanMealsRssCacheAll()
	CleanMealsEpgCacheAll()
}

func CleanMealsCacheRebuildOne(id int64) {
	log.Println("删除套餐订阅缓存: ", id)
	dao.Cache.Delete("rssMealTxt_" + strconv.FormatInt(id, 10))
	dao.Cache.Delete("rssMealM3u8_" + strconv.FormatInt(id, 10))
	dao.Cache.Delete("mytvMeal*")
	CleanMealsXmlCacheOne(id)
}

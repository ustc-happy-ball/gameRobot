package main

import (
	"fmt"
	"os"
	"strconv"

	pb "github.com/LILILIhuahuahua/ustc_tencent_game/api/proto"
	"github.com/LILILIhuahuahua/ustc_tencent_game/configs"
	"github.com/LILILIhuahuahua/ustc_tencent_game/framework/event"
	event2 "github.com/LILILIhuahuahua/ustc_tencent_game/gameInternal/event"
	"github.com/LILILIhuahuahua/ustc_tencent_game/gameInternal/event/info"
	"github.com/LILILIhuahuahua/ustc_tencent_game/gameInternal/event/notify"
	"github.com/LILILIhuahuahua/ustc_tencent_game/gameInternal/event/request"
	"github.com/LILILIhuahuahua/ustc_tencent_game/gameInternal/event/response"

	// "github.com/LILILIhuahuahua/ustc_tencent_game/gameInternal/game"
	"log"
	"math/rand"
	"time"

	"github.com/LILILIhuahuahua/ustc_tencent_game/model"
	"github.com/LILILIhuahuahua/ustc_tencent_game/tools"
	"github.com/golang/protobuf/proto"
	"github.com/xtaci/kcp-go"
)

const (
	remoteAddr = "150.158.216.120:32001"
	localAddr  = "127.0.0.1:8888"
)

type Robot struct {
	recvQueue *event.EventRingQueue
	sessionId int32
	hero      *model.Hero
	session   *kcp.UDPSession
	done      chan struct{}
	index     int
	file      *os.File
	quit      chan struct{}
}

func NewTestRobot(idx int) *Robot {
	seq := strconv.Itoa(idx)
	f, err := os.OpenFile("./data/"+seq+".txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0777)
	if err != nil {
		log.Fatalln(err)
	}
	return &Robot{
		recvQueue: event.NewEventRingQueue(500),
		sessionId: tools.UUID_UTIL.GenerateInt32UUID(),
		done:      make(chan struct{}),
		index:     idx,
		file:      f,
		quit: make(chan struct{}),
	}
}

func (robot *Robot) boot() {
	go robot.handle()
	robot.accept()
}

func (robot *Robot) accept() {
	if sess, err := kcp.DialWithOptions(remoteAddr, nil, 0, 0); err == nil {
		//sess调优
		sess.SetNoDelay(1, 10, 2, 1)
		sess.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(2)))
		sess.SetACKNoDelay(true)
		robot.session = sess
		//开启进入世界流程
		data := request.NewEnterGameRequest(robot.sessionId, *info.NewConnectInfo("0.0.0.0", -1), "").ToGMessageBytes()
		sess.Write(data)
		buf := make([]byte, 4096)
		for {
			num, _ := sess.Read(buf)
			if num > 0 {
				pbGMsg := &pb.GMessage{}
				proto.Unmarshal(buf[:num], pbGMsg)
				msg := event2.GMessage{}
				m := msg.CopyFromMessage(pbGMsg)
				robot.recvQueue.Push(m)
			}
		}
	}
}

func (robot *Robot) handle() {
	for {
		select {
		case <-robot.done:
			robot.file.WriteString("[Handler]收到关闭信号，退出handle处理程序，停止从队列中消费GMessage消息！")
			return
		default:
			e, err := robot.recvQueue.Pop()
			if nil == e { //todo
				continue
			}
			if nil != err {
				log.Println(err)
				continue
			}
			msg := e.(*event2.GMessage)
			if nil == msg.Data {
				robot.file.WriteString(fmt.Sprintf("[未知包无法处理]收到机器人还不能处理的消息包！GMessage：%+v \n", msg))
				//log.Printf()
				continue
			}
			robot.dispatchGMessage(msg)
		}
	}
}

func (robot *Robot) dispatchGMessage(msg *event2.GMessage) {
	// 二级解码
	data := msg.Data
	switch data.GetCode() {

	case int32(pb.GAME_MSG_CODE_ENTER_GAME_RESPONSE):
		robot.onEnterGameResponse(data.(*response.EnterGameResponse))

	case int32(pb.GAME_MSG_CODE_ENTITY_INFO_CHANGE_NOTIFY):
		robot.onEntityInfoChange(data.(*notify.EntityInfoChangeNotify))

	case int32(pb.GAME_MSG_CODE_ENTITY_INFO_NOTIFY):
		robot.onEntityInfoChange(data.(*notify.EntityInfoChangeNotify))

	case int32(pb.GAME_MSG_CODE_GAME_FINISH_NOTIFY):
		robot.close()
	case int32(pb.GAME_MSG_CODE_ENTER_GAME_NOTIFY):
		robot.onEnterGameNotify(data.(*notify.EnterGameNotify))
	case int32(pb.GAME_MSG_CODE_GAME_RANK_LIST_NOTIFY):
		robot.onGameRankListNotify(data.(*notify.GameRankListNotify))
	case int32(pb.GAME_MSG_CODE_GAME_GLOBAL_INFO_NOTIFY):
		robot.onGameGlobalInfoNotify(data.(*notify.GameGlobalInfoNotify))
	case int32(pb.GAME_MSG_CODE_ENTITY_INFO_CHANGE_RESPONSE):
		robot.onEntityInfoChangeResponse(data.(*response.EntityInfoChangeResponse))
	case int32(pb.GAME_MSG_CODE_HERO_VIEW_NOTIFY):
		robot.onHeroViewNotify(data.(*notify.HeroViewNotify))
	case int32(pb.GAME_MSG_CODE_TIME_NOTIFY):
	default:
		robot.file.WriteString(fmt.Sprintf("Unrecognized message code: %+v, %+v\n", data.GetCode(), data))
	}
}

func (robot *Robot) onHeioQuit(rsp *response.HeroQuitResponse) {
	robot.file.WriteString(fmt.Sprintf("[HeroQuitResponse] %+v\n", rsp))
}
func (robot *Robot) onHeroViewNotify(nty *notify.HeroViewNotify) {
	robot.file.WriteString(fmt.Sprintf("[HeroViewNotify] %+v\n", nty))
}

func (robot *Robot) onEntityInfoChangeResponse(rsp *response.EntityInfoChangeResponse) {
	robot.file.WriteString(fmt.Sprintf("[EntifyInfoChangeResponse] %+v\n", rsp))
}

func (robot *Robot) onGameGlobalInfoNotify(nty *notify.GameGlobalInfoNotify) {
	robot.file.WriteString(fmt.Sprintf("[GameGlobalInfoNotify] %+v\n", nty))
}

func (robot *Robot) onGameRankListNotify(nty *notify.GameRankListNotify) {
	robot.file.WriteString(fmt.Sprintf("[GameRankListNotify] %+v\n", nty))
}

func (robot *Robot) onEnterGameNotify(nty *notify.EnterGameNotify) {
	robot.file.WriteString(fmt.Sprintf("[EnterGameNotify] %+v\n", nty))
}

func (robot *Robot) onEnterGameResponse(resp *response.EnterGameResponse) {
	robot.hero = model.NewHero("", nil)
	robot.hero.ID = resp.HeroId
	xr := configs.MapMaxX - configs.MapMinX
	yr := configs.MapMaxY - configs.MapMinY
	rand.Seed(time.Now().Unix())
	randX := rand.Int31n(int32(xr)) + int32(configs.MapMinX)
	randY := rand.Int31n(int32(yr)) + int32(configs.MapMinY)
	robot.hero.HeroPosition.X = float32(randX)
	robot.hero.HeroPosition.Y = float32(randY)
	// 开启本地更新英雄位置线程
	go robot.updateHeroPos()
	go robot.updateHeroDirt()
}

func (robot *Robot) onEntityInfoChange(notify *notify.EntityInfoChangeNotify) {
	if notify.EntityType == int32(pb.ENTITY_TYPE_HERO_TYPE) && notify.EntityId == robot.hero.ID { //只处理自己的状态
		robot.file.WriteString(fmt.Sprintf("[EntityInfoChange]收到本机器人持有英雄状态改变推送！位置：%+v, 方向：%+v, 信息为：%+v", notify.HeroMsg.HeroPosition, notify.HeroMsg.HeroDirection, notify.HeroMsg))
		heroInfo := notify.HeroMsg
		if heroInfo.Status == int32(pb.HERO_STATUS_DEAD) {
			robot.file.WriteString(fmt.Sprintln("[robot]英雄阵亡！"))
			robot.close()
		}
		robot.updateHero(heroInfo)
	}
}

func (robot *Robot) updateHero(heroInfo *info.HeroInfo) {
	if nil != heroInfo.HeroPosition {
		robot.hero.HeroPosition.X = heroInfo.HeroPosition.CoordinateX
		robot.hero.HeroPosition.Y = heroInfo.HeroPosition.CoordinateY
	}
	if nil != heroInfo.HeroDirection {
		robot.hero.HeroDirection.X = heroInfo.HeroDirection.CoordinateX
		robot.hero.HeroDirection.Y = heroInfo.HeroDirection.CoordinateY
	}
	robot.hero.Status = heroInfo.Status
	robot.hero.Score = heroInfo.Score
	robot.hero.Size = heroInfo.Size
	robot.hero.Invincible = heroInfo.Invincible
	robot.hero.Speed = heroInfo.Speed
	robot.hero.SpeedDown = heroInfo.SpeedDown
	robot.hero.SpeedUp = heroInfo.SpeedUp
}

// TODO 循环优化
func (robot *Robot) updateHeroPos() {
	for {
		select {
		case <-robot.done:
			robot.file.WriteString(fmt.Sprintln("Exit updateHeroPos..."))
			return
		default:
			//fmt.Println("[go updateHeroPos]更新位置")
			nowTime := time.Now().UnixNano()
			// 更新玩家位置
			timeElapse := nowTime - robot.hero.UpdateTime
			robot.hero.UpdateTime = nowTime
			distance := float64(timeElapse) * float64(robot.hero.Speed) / 1e9
			x, y := tools.CalXY(distance, robot.hero.HeroDirection.X, robot.hero.HeroDirection.Y)
			robot.hero.HeroPosition.X += x
			robot.hero.HeroPosition.X = tools.GetMax(robot.hero.HeroPosition.X, configs.MapMinX+50.0)
			robot.hero.HeroPosition.X = tools.GetMin(robot.hero.HeroPosition.X, configs.MapMaxX-50.0)
			robot.hero.HeroPosition.Y += y
			robot.hero.HeroPosition.Y = tools.GetMax(robot.hero.HeroPosition.Y, configs.MapMinY+50.0)
			robot.hero.HeroPosition.Y = tools.GetMin(robot.hero.HeroPosition.Y, configs.MapMaxY-50.0)
			time.Sleep(2 * time.Second) //睡 2s, 模拟真实操作
		}
	}
}

// TODO 循环优化
func (robot *Robot) updateHeroDirt() {
	for {
		select {
		case <-robot.done:
			robot.file.WriteString(fmt.Sprintln("Exit updateHeroDirt..."))
			return
		default:
			rand.Seed(time.Now().UnixNano())
			//sleepTime := rand.Int63()%5
			randDict := rand.Intn(4)
			switch randDict {
			case 0:
				robot.hero.HeroDirection.X = 1
				robot.hero.HeroDirection.Y = 0
			case 1:
				robot.hero.HeroDirection.X = -1
				robot.hero.HeroDirection.Y = 0
			case 2:
				robot.hero.HeroDirection.X = 0
				robot.hero.HeroDirection.Y = 1
			case 3:
				robot.hero.HeroDirection.X = 0
				robot.hero.HeroDirection.Y = -1
			}
			heroInfo := info.NewHeroInfo(robot.hero)
			entityInfoChangeReq := request.NewEntityInfoChangeRequest(int32(pb.EVENT_TYPE_HERO_MOVE), robot.hero.ID, -1, "", *heroInfo)
			data := entityInfoChangeReq.ToGMessageBytes()
			robot.session.Write(data)
			time.Sleep(2 * time.Second) // 每 2s 改变一次方向
		}
	}
}

func (robot *Robot) close() {
	//英雄阵亡，处理机器人资源回收
	robot.file.WriteString(fmt.Sprintln("[robot]关闭并回收机器人资源!"))
	close(robot.done)
	robot.file.Close()
	time.Sleep(4 * time.Second)
}

func (robot *Robot) HeroQuitGame() {
	msg := pb.GMessage{
		MsgType:   pb.MSG_TYPE_REQUEST,
		MsgCode:   pb.GAME_MSG_CODE_HERO_QUIT_REQUEST,
		SessionId: robot.sessionId,
		SeqId:     0,
		Request:   &pb.Request{HeroQuitRequest: &pb.HeroQuitRequest{
			HeroId: robot.hero.ID,
			Time:   time.Now().UnixNano(),
		}},
		SendTime:  time.Now().UnixNano(),
	}
	data,err := proto.Marshal(&msg)
	if err != nil {
		log.Println(err)
	}
	robot.session.Write(data)
}

func BootRobot(idx int, done chan struct{}, quit chan os.Signal) {
	robot := NewTestRobot(idx)
	go robot.boot()

	t := time.Tick(2 * time.Second)
	for {
		select {
		case <-robot.done:
			close(done)
			time.Sleep(1 * time.Second)
			return
		case <- quit:
			robot.HeroQuitGame()
			close(done)
			//fmt.Printf("Robot %d interpret...", robot.index)
			time.Sleep(1 * time.Second)
			return
		case <-t:
			time.Sleep(1 * time.Second)
		}
	}
}

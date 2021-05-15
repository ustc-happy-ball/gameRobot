package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/LILILIhuahuahua/ustc_tencent_game/gameInternal/game"
)

var robotNum int = 1
var interval int

func init() {
	g := &game.GameStarter{}
	g.Init()
}

func main() {
	flag.IntVar(&robotNum, "n", 1, "Robot number")
	flag.IntVar(&interval,"t",0,"Interval to generate a bot")
	flag.Parse()

	// robot quit gracefully
	quitSig := make(chan os.Signal)
	signal.Notify(quitSig,syscall.SIGINT, syscall.SIGTERM)
	go gracefulQuit(quitSig)

	doneCh := make([]chan struct{}, robotNum)
	done := make(chan struct{})
	cases := make([]reflect.SelectCase, robotNum)
	// producer
	for i := range doneCh {
		doneCh[i] = make(chan struct{})
		go BootRobot(i, doneCh[i],quitSig) // why go build main.go can't find BootRoot function, unless you run go build main.go robot.go
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(doneCh[i])}
		time.Sleep(20 * time.Millisecond)
	}

	// recover
	defer func() {
		if err := recover(); err != nil {
			close(quitSig)
			time.Sleep(2 * time.Second)
		}
	}()

	time.Sleep(2 * time.Second)
	// consumer
	go func() {
		var count int
		fmt.Printf("----------------------------%d Robots--------------------------------\n", robotNum)
		remaining := len(cases)
		for remaining > 0 {
			chosen, _, ok := reflect.Select(cases)
			if !ok {
				cases[chosen].Chan = reflect.ValueOf(nil)
				count++
				fmt.Printf("[%d/%d] Robot %d quit the game\n", count, robotNum, chosen)
				remaining -= 1
				continue
			}
		}

		done <- struct{}{}
	}()

	<-done
}


func gracefulQuit(quitSig chan os.Signal) {
	<- quitSig
	fmt.Println()
	fmt.Println("Receive SIGINT signal, quit...")
	close(quitSig)
	time.Sleep(3 * time.Second)
}

// 从一个closed 的 channel 中读取数据，读到的数据默认是零值

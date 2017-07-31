// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin linux windows

// An app that draws a green triangle on a red background.
//
// Note: This demo is an early preview of Go 1.5. In order to build this
// program as an Android APK using the gomobile tool.
//
// See http://godoc.org/golang.org/x/mobile/cmd/gomobile to install gomobile.
//
// Get the basic example and use gomobile to build or install it on your device.
//
//   $ go get -d golang.org/x/mobile/example/basic
//   $ gomobile build golang.org/x/mobile/example/basic # will build an APK
//
//   # plug your Android device to your computer or start an Android emulator.
//   # if you have adb installed on your machine, use gomobile install to
//   # build and deploy the APK to an Android target.
//   $ gomobile install golang.org/x/mobile/example/basic
//
// Switch to your device or emulator to start the Basic application from
// the launcher.
// You can also run the application on your desktop by running the command
// below. (Note: It currently doesn't work on Windows.)
//   $ go install golang.org/x/mobile/example/basic && basic
package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	stdlog "log"
	"net"
	"os"
	"time"

	"github.com/nfnt/resize"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/exp/app/debug"
	"golang.org/x/mobile/exp/gl/glutil"
	"golang.org/x/mobile/gl"
)

type touchEvent struct {
	x, y int16
	t    touch.Type
}

const (
	serverAddr                   = "192.168.25.9:2501"
	defaultPointerVelocityFactor = 4
)

var (
	log      *stdlog.Logger
	images   *glutil.Images
	fps      *debug.FPS
	program  gl.Program
	position gl.Attrib
	offset   gl.Uniform
	color    gl.Uniform
	bunnySrc image.Image
	bunny    *Bunny

	green    float32
	touchX   float32
	touchY   float32
	x0, y0   int16
	tapBegin time.Time
	evtChan  chan touchEvent
)

func init() {
	log = stdlog.New(os.Stdout, "mouseapp: ", stdlog.Flags())
}

func main() {
	evtChan = make(chan touchEvent, 256)

	app.Main(func(a app.App) {
		var glctx gl.Context
		var sz size.Event
		for e := range a.Events() {
			switch e := a.Filter(e).(type) {
			case lifecycle.Event:
				switch e.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					glctx, _ = e.DrawContext.(gl.Context)
					onStart(glctx)
					a.Send(paint.Event{})
				case lifecycle.CrossOff:
					onStop(glctx)
					glctx = nil
				}
			case size.Event:
				sz = e
				//				touchX = float32(sz.WidthPx / 2)
				//				touchY = float32(sz.HeightPx / 2)
			case paint.Event:
				if glctx == nil || e.External {
					// As we are actively painting as fast as
					// we can (usually 60 FPS), skip any paint
					// events sent by the system.
					continue
				}

				onPaint(glctx, sz)
				a.Publish()
				// Drive the animation by preparing to paint the next frame
				// after this one is shown.
				a.Send(paint.Event{})
			case touch.Event:
				touchX = e.X / sz.PixelsPerPt
				touchY = e.Y / sz.PixelsPerPt
				evtChan <- touchEvent{
					x: int16(touchX),
					y: int16(touchY),
					t: e.Type,
				}
			}
		}
	})
}

func onStart(glctx gl.Context) {
	err := remoteControl()
	if err != nil {
		log.Fatal(err)
	}

	f, err := asset.Open("bunny.jpg")
	if err != nil {
		log.Fatal(err)
	}
	bunnyOrig, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	f.Close()

	bunnySrc = resize.Resize(50, 0, bunnyOrig, resize.Lanczos3)

	program, err = glutil.CreateProgram(glctx, vertexShader, fragmentShader)
	if err != nil {
		log.Printf("error creating GL program: %v", err)
		return
	}

	images = glutil.NewImages(glctx)
	fps = debug.NewFPS(images)
	bunny = NewBunny(images, bunnySrc)
}

func onStop(glctx gl.Context) {
	glctx.DeleteProgram(program)
	fps.Release()
	bunny.Release()
	images.Release()
}

func onPaint(glctx gl.Context, sz size.Event) {
	glctx.ClearColor(1, 1, 1, 1)
	glctx.Clear(gl.COLOR_BUFFER_BIT)

	glctx.UseProgram(program)

	fps.Draw(sz)
	bunny.Draw(sz, touchX, touchY)
}

const vertexShader = `#version 100
void main() {

}`

const fragmentShader = `#version 100
void main() {

}`

func remoteControl() error {
	sAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return err
	}

	//lAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	//if err != nil {
	//return err
	//}

	conn, err := net.DialUDP("udp", nil, sAddr)
	if err != nil {
		return err
	}

	log.Printf("Established connection to %s \n", serverAddr)
	log.Printf("Remote UDP address : %s \n", conn.RemoteAddr().String())
	log.Printf("Local UDP client address : %s \n", conn.LocalAddr().String())

	go sendEvents(conn)
	return nil
}

func sendEvents(conn net.Conn) {
	for e := range evtChan {
		if e.t == touch.TypeBegin {
			x0 = e.x
			y0 = e.y
			tapBegin = time.Now()
			continue
		} else if e.t == touch.TypeEnd {
			since := time.Since(tapBegin)
			if since < 1*time.Second {
				_, err := conn.Write([]byte("click"))
				if err != nil {
					log.Printf("err sending data: %s", err.Error())
				}
				continue
			}
		}

		x := e.x - x0
		y := e.y - y0

		x *= defaultPointerVelocityFactor
		y *= defaultPointerVelocityFactor

		x0 = e.x
		y0 = e.y

		data := fmt.Sprintf("mouse: %d,%d", int(x), int(y))
		_, err := conn.Write([]byte(data))
		if err != nil {
			log.Printf("err sending data: %s", err.Error())
		}
	}
}

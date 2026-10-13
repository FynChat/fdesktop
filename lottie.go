package main

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// LottiePlayer is a small looping animation. Fyne has no Lottie runtime, so
// this stands in for one: a shape whose size or colour is driven by a
// fyne.Animation that reverses instead of restarting. Reversing is what keeps
// the loop on a single goroutine — a Start/Sleep/Start cycle leaks one
// goroutine per animation because nothing ever stops the sleep.
type LottiePlayer struct {
	container *fyne.Container
	animation *fyne.Animation
}

// NewLottiePlayer returns a dot that pulses between two sizes.
func NewLottiePlayer() *LottiePlayer {
	dot := canvas.NewCircle(theme.ForegroundColor())
	dot.Resize(fyne.NewSize(20, 20))

	box := container.NewCenter(dot)
	player := &LottiePlayer{container: box}

	const pulse = 0.4
	player.animation = fyne.NewAnimation(1000*time.Millisecond, func(v float32) {
		// v runs 0 → 1; the scale peaks at the middle of the cycle so the
		// dot grows and shrinks rather than snapping back.
		scale := 1 + pulse*v
		size := 20 * scale
		dot.Resize(fyne.NewSize(size, size))
		dot.Refresh()
	})
	player.animation.AutoReverse = true
	player.animation.RepeatCount = fyne.AnimationRepeatForever
	player.animation.Start()

	return player
}

// Container is the animation's canvas object, ready to place in a layout.
func (l *LottiePlayer) Container() fyne.CanvasObject {
	return l.container
}

// Stop halts the loop so the animation does not outlive the view holding it.
func (l *LottiePlayer) Stop() {
	if l.animation != nil {
		l.animation.Stop()
	}
}

// AnimatedIcon returns a square that fades in and out, useful as a busy hint.
func AnimatedIcon() fyne.CanvasObject {
	rect := canvas.NewRectangle(color.NRGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF})
	rect.CornerRadius = 2

	anim := fyne.NewAnimation(1000*time.Millisecond, func(v float32) {
		rect.FillColor = color.NRGBA{R: 0x3B, G: 0x82, B: 0xF6, A: uint8(128 + 127*v)}
		rect.Refresh()
	})
	anim.AutoReverse = true
	anim.RepeatCount = fyne.AnimationRepeatForever
	anim.Start()

	return container.NewCenter(rect)
}

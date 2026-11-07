// Confetti and sparks (simulated here, drawn in board.go). (c) 2026 Ironite
package main

import (
	"image/color"
	"math"
	"math/rand"
)

type particle struct {
	x, y, vx, vy, life, age, w, h, rot, vr float64
	col                                    color.NRGBA
}

func (u *UI) burst(x, y float64, n int, cols []color.NRGBA, speed, life float64) {
	for i := 0; i < n; i++ {
		ang := rand.Float64() * 2 * math.Pi
		sp := speed * (.3 + rand.Float64()*.7)
		u.parts = append(u.parts, &particle{x: x, y: y, vx: math.Cos(ang) * sp, vy: math.Sin(ang)*sp - speed*.35,
			life: life * (.6 + rand.Float64()*.4), w: 6 + rand.Float64()*6, h: 4 + rand.Float64()*4, rot: rand.Float64() * 6, vr: (rand.Float64() - .5) * 12,
			col: cols[rand.Intn(len(cols))]})
	}
}

func (u *UI) confetti() {
	u.burst(BX+FR/2, BY+FR*.45, 150, []color.NRGBA{Hex("#eac46a"), Hex("#f7ecd4"), Hex("#8e2a1e"), Hex("#2f4a33"), Hex("#c2241d"), Hex("#1d1d22")}, 720, 2.6)
}

func (u *UI) crownBurst(s int) {
	x, y := u.a.sqXY(s)
	u.burst(BX+x+SQ/2, BY+y+SQ/2, 40, []color.NRGBA{Hex("#f5d277"), Hex("#fff3c4"), Hex("#e3bd5a")}, 320, 1.2)
}

func (u *UI) stepParticles(dt float64) {
	keep := u.parts[:0]
	for _, p := range u.parts {
		p.age += dt
		if p.age >= p.life {
			continue
		}
		p.vy += 900 * dt
		p.vx *= math.Pow(.4, dt)
		p.x += p.vx * dt
		p.y += p.vy * dt
		p.rot += p.vr * dt
		keep = append(keep, p)
	}
	u.parts = keep
}

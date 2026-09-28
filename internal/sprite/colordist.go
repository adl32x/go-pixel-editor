package sprite

import (
	"math"
	"sync"
)

// colorDistance is the perceptual difference between two "#rrggbb" colors:
// CIEDE2000 ΔE over CIELAB (D65). Plain RGB distance picks visibly wrong
// matches — e.g. it maps a teal onto a muddy grey-green rather than the
// sage the eye reads as closest — because equal RGB steps aren't equal
// perceived steps. A malformed color sorts as maximally distant rather than
// erroring, so a single bad palette entry can't break every lookup.
func colorDistance(a, b string) float64 {
	la, aok := hexToLab(a)
	lb, bok := hexToLab(b)
	if !aok || !bok {
		return math.MaxFloat64
	}
	return ciede2000(la, lb)
}

type lab struct{ L, A, B float64 }

// labCache memoizes hex -> Lab: ColorToChar compares every pixel against
// every palette entry, and the same few dozen colors come up over and over.
var labCache sync.Map // string -> lab

func hexToLab(hex string) (lab, bool) {
	if v, ok := labCache.Load(hex); ok {
		return v.(lab), true
	}
	r, g, b, ok := parseHexColor(hex)
	if !ok {
		return lab{}, false
	}
	l := rgbToLab(r, g, b)
	labCache.Store(hex, l)
	return l, true
}

// rgbToLab converts 8-bit sRGB to CIELAB with a D65 white point.
func rgbToLab(r, g, b int) lab {
	linear := func(c int) float64 {
		v := float64(c) / 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	rl, gl, bl := linear(r), linear(g), linear(b)
	x := (0.4124564*rl + 0.3575761*gl + 0.1804375*bl) / 0.95047
	y := 0.2126729*rl + 0.7151522*gl + 0.0721750*bl
	z := (0.0193339*rl + 0.1191920*gl + 0.9503041*bl) / 1.08883
	f := func(t float64) float64 {
		if t > 216.0/24389 {
			return math.Cbrt(t)
		}
		return (24389.0/27*t + 16) / 116
	}
	fx, fy, fz := f(x), f(y), f(z)
	return lab{L: 116*fy - 16, A: 500 * (fx - fy), B: 200 * (fy - fz)}
}

// ciede2000 is the CIEDE2000 color difference (kL = kC = kH = 1), per
// Sharma, Wu & Dalal, "The CIEDE2000 Color-Difference Formula" (2005).
func ciede2000(c1, c2 lab) float64 {
	const deg = math.Pi / 180

	cab := (math.Hypot(c1.A, c1.B) + math.Hypot(c2.A, c2.B)) / 2
	cab7 := math.Pow(cab, 7)
	g := 0.5 * (1 - math.Sqrt(cab7/(cab7+math.Pow(25, 7))))
	a1, a2 := (1+g)*c1.A, (1+g)*c2.A
	cp1, cp2 := math.Hypot(a1, c1.B), math.Hypot(a2, c2.B)

	hue := func(b, a float64) float64 {
		if b == 0 && a == 0 {
			return 0
		}
		h := math.Atan2(b, a) / deg
		if h < 0 {
			h += 360
		}
		return h
	}
	hp1, hp2 := hue(c1.B, a1), hue(c2.B, a2)

	dL := c2.L - c1.L
	dC := cp2 - cp1
	var dh float64
	switch {
	case cp1*cp2 == 0:
		dh = 0
	case math.Abs(hp2-hp1) <= 180:
		dh = hp2 - hp1
	case hp2-hp1 > 180:
		dh = hp2 - hp1 - 360
	default:
		dh = hp2 - hp1 + 360
	}
	dH := 2 * math.Sqrt(cp1*cp2) * math.Sin(dh/2*deg)

	lp := (c1.L + c2.L) / 2
	cp := (cp1 + cp2) / 2
	var hp float64
	switch {
	case cp1*cp2 == 0:
		hp = hp1 + hp2
	case math.Abs(hp1-hp2) <= 180:
		hp = (hp1 + hp2) / 2
	case hp1+hp2 < 360:
		hp = (hp1 + hp2 + 360) / 2
	default:
		hp = (hp1 + hp2 - 360) / 2
	}

	t := 1 - 0.17*math.Cos((hp-30)*deg) + 0.24*math.Cos(2*hp*deg) +
		0.32*math.Cos((3*hp+6)*deg) - 0.20*math.Cos((4*hp-63)*deg)
	dTheta := 30 * math.Exp(-math.Pow((hp-275)/25, 2))
	cp7 := math.Pow(cp, 7)
	rc := 2 * math.Sqrt(cp7/(cp7+math.Pow(25, 7)))
	sl := 1 + 0.015*math.Pow(lp-50, 2)/math.Sqrt(20+math.Pow(lp-50, 2))
	sc := 1 + 0.045*cp
	sh := 1 + 0.015*cp*t
	rt := -math.Sin(2*dTheta*deg) * rc

	lTerm, cTerm, hTerm := dL/sl, dC/sc, dH/sh
	return math.Sqrt(lTerm*lTerm + cTerm*cTerm + hTerm*hTerm + rt*cTerm*hTerm)
}

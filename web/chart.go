package web

import (
	"fmt"
	"math"
	"strings"

	"github.com/pedro-git-projects/gymtracker/model"
)

// chart is a server rendered single series line chart. All coordinates are
// preformatted so the template only has to place them.
type chart struct {
	Title  string
	W, H   int
	Left   string
	Right  string
	Top    string
	Bottom string
	TipY   string
	Path   string
	Ticks  []chartTick
	Points []chartPoint
	Dots   bool
	First  string
	Last   string
	EndX   string
	EndY   string
	End    string
}

type chartTick struct {
	Y     string
	Label string
}

type chartPoint struct {
	X, Y   string
	HitX   string
	HitW   string
	Tip    string
	TipX   string
	Anchor string
}

const (
	chartW      = 420
	chartH      = 200
	chartLeft   = 34.0
	chartRight  = 380.0
	chartTop    = 26.0
	chartBottom = 172.0
	maxDots     = 40
)

func px(v float64) string { return fmt.Sprintf("%.1f", v) }

// niceStep rounds a raw tick step up to 1, 2 or 5 times a power of ten,
// never going below 1 because loads and reps are whole numbers.
func niceStep(raw float64) float64 {
	if raw <= 1 {
		return 1
	}
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range []float64{1, 2, 5, 10} {
		if raw <= m*mag {
			return m * mag
		}
	}
	return 10 * mag
}

// progressionChart plots the top load of every session in entries, oldest
// first. Exercises that were never loaded plot the best set's reps instead.
// It returns nil when there are fewer than two sessions to compare.
func progressionChart(entries []model.ExerciseEntry, bodyweight bool) *chart {
	if len(entries) < 2 {
		return nil
	}
	byReps := true
	for _, e := range entries {
		if e.Log.TopLoad() > 0 {
			byReps = false
			break
		}
	}
	title, unit := "Top load per session (kg)", "kg"
	switch {
	case byReps:
		title, unit = "Best set per session (reps)", " reps"
	case bodyweight:
		title = "Top added load per session (kg)"
	}

	values := make([]float64, len(entries))
	lo, hi := math.Inf(1), math.Inf(-1)
	for i, e := range entries {
		v := float64(e.Log.TopLoad())
		if byReps {
			v = float64(e.Log.TopReps())
		}
		values[i] = v
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	step := niceStep((hi - lo) / 3)
	lo = math.Floor(lo/step) * step
	hi = math.Ceil(hi/step) * step
	if hi == lo {
		hi = lo + step
		if lo >= step {
			lo -= step
		}
	}

	c := &chart{
		Title: title,
		W:     chartW, H: chartH,
		Left: px(chartLeft), Right: px(chartRight),
		Top: px(chartTop), Bottom: px(chartBottom),
		TipY:  px(chartTop - 10),
		Dots:  len(entries) <= maxDots,
		First: entries[0].Date.Format("02 Jan 2006"),
		Last:  entries[len(entries)-1].Date.Format("02 Jan 2006"),
	}
	yOf := func(v float64) float64 {
		return chartBottom - (v-lo)/(hi-lo)*(chartBottom-chartTop)
	}
	for v := lo; v <= hi+step/2; v += step {
		c.Ticks = append(c.Ticks, chartTick{Y: px(yOf(v)), Label: fmt.Sprintf("%.0f", v)})
	}

	gap := (chartRight - chartLeft) / float64(len(entries)-1)
	var path strings.Builder
	for i, e := range entries {
		x, y := chartLeft+gap*float64(i), yOf(values[i])
		if i == 0 {
			fmt.Fprintf(&path, "M%.1f %.1f", x, y)
		} else {
			fmt.Fprintf(&path, " L%.1f %.1f", x, y)
		}
		p := chartPoint{
			X: px(x), Y: px(y),
			HitX: px(x - gap/2), HitW: px(gap),
			Tip:  fmt.Sprintf("%s · %.0f%s", e.Date.Format("Mon 02 Jan 2006"), values[i], unit),
			TipX: px(x), Anchor: "middle",
		}
		// Keep the readout inside the frame near either edge.
		switch {
		case x < chartLeft+90:
			p.TipX, p.Anchor = px(chartLeft), "start"
		case x > chartRight-90:
			p.TipX, p.Anchor = px(chartRight), "end"
		}
		c.Points = append(c.Points, p)
	}
	c.Path = path.String()
	last := c.Points[len(c.Points)-1]
	c.EndX, c.EndY = px(chartRight+8), last.Y
	c.End = fmt.Sprintf("%.0f", values[len(values)-1])
	return c
}

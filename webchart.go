package main

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func legendLabel(s Series) string {
	if len(s.Y) == 0 {
		return s.Label
	}

	st := int(s.Y[0])
	en := int(s.Y[len(s.Y)-1])
	if st != 0 {
		return fmt.Sprintf("%s (%d → %d)", s.Label, st, en)
	}

	return fmt.Sprintf("%s (%d)", s.Label, en)
}

// htmlPalette derives from the terminal palette's light-mode shades.
// HTML renders on a fixed light background so adaptive colors are not applicable.
var htmlPalette = func() opts.Colors {
	c := make(opts.Colors, len(palette))
	for i, ac := range palette {
		c[i] = ac.Light
	}
	return c
}()

// manual ECharts init on a plain div — avoids go-echarts wrapper breaking the flex layout.
var pageTpl = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>git-trend</title>
  {{- range .Scripts }}
  <script src="{{ . }}"></script>
  {{- end }}
  <style>
    *, *::before, *::after { box-sizing: border-box; }
    html, body { margin: 0; padding: 0; height: 100%; font-family: system-ui, -apple-system, sans-serif; }
    body { display: flex; flex-direction: column; }
    #chart { flex: 1; min-height: 0; }
  </style>
</head>
<body>
  <div id="chart"></div>
  <script>
    var chart = echarts.init(document.getElementById('chart'));
    chart.setOption({{ .Option }});
    window.addEventListener('resize', function() { chart.resize(); });
  </script>
</body>
</html>
`))

type pageData struct {
	Scripts []string
	Option  template.JS
}

func RenderHTML(s []Series, samples []time.Time, start, end time.Time, name string, w io.Writer) error {
	line := charts.NewLine()

	title := name
	if title == "" {
		title = "git-trend"
	}
	subtitle := fmt.Sprintf("cumulative commits  %s  →  %s",
		start.Format(time.DateOnly), end.Format(time.DateOnly))

	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    title,
			Subtitle: subtitle,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger: "axis",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:  "slider",
			Start: 0,
			End:   100,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
			Type: "scroll",
			// Position at bottom, above the datazoom slider.
			Bottom: "45",
		}),
		// plain numbers — ECharts ignores "px".
		charts.WithGridOpts(opts.Grid{
			Top:    "70",
			Bottom: "100",
			Left:   "60",
			Right:  "20",
		}),
		charts.WithColorsOpts(htmlPalette),
	)

	if len(s) > 0 {
		xLabels := make([]string, len(samples))
		for i, t := range samples {
			xLabels[i] = t.Format(time.DateOnly)
		}
		line.SetXAxis(xLabels)
	}

	for _, ser := range s {
		items := make([]opts.LineData, len(ser.Y))
		for i, v := range ser.Y {
			item := opts.LineData{Value: v}
			// Only show a dot where the value changed — one dot per commit change.
			if i == 0 || v == ser.Y[i-1] {
				item.Symbol = "none"
			}
			items[i] = item
		}
		line.AddSeries(legendLabel(ser), items)
	}

	// triggers Validate to set up ChartID/assets; rendered element is discarded.
	_ = line.RenderSnippet()
	assets := line.GetAssets()

	return pageTpl.Execute(w, pageData{
		Scripts: assets.JSAssets.Values,
		Option:  template.JS(line.JSONNotEscaped()),
	})
}

// writeChart builds and writes an HTML chart file.
// When out is empty a temporary file is created and its path returned.
func writeChart(opts BuildOptions, since, until, out string) (string, error) {
	commits, err := Log(Options{Since: since, Until: until})
	if err != nil {
		return "", err
	}

	if len(commits) == 0 {
		return "", fmt.Errorf("no commits found for the given filters")
	}

	opts.Width = min(len(commits), 500)
	chart, err := Build(commits, opts)
	if err != nil {
		return "", err
	}

	if chart.Dropped.Authors > 0 {
		fmt.Fprintf(os.Stderr, "git-trend: warning: --author matched %d authors, showing the top %d by commit count\n", len(chart.Series)+chart.Dropped.Authors, len(chart.Series))
		fmt.Fprintln(os.Stderr, "git-trend: hint: refine --author to narrow the selection")
	}

	if chart.Dropped.Total {
		fmt.Fprintf(os.Stderr, "git-trend: warning: no slot for --total (showing %d author lines)\n", len(chart.Series))
	}

	var f *os.File
	var path string
	if out == "" {
		f, err = os.CreateTemp("", "git-trend-*.html")
		if err != nil {
			return "", err
		}
		path = f.Name()
	} else {
		path = out
		f, err = os.Create(path)
		if err != nil {
			return "", err
		}
	}
	defer f.Close()

	if err := RenderHTML(chart.Series, chart.Samples, chart.Start, chart.End, repoName(), f); err != nil {
		return "", err
	}

	return path, nil
}

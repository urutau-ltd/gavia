// @license magnet:?xt=urn:btih:0b31508aeb0634b347b8270c7bee4d411b5d4109&dn=agpl-3.0.txt AGPL-3.0-or-later
const uptimeChartDataElement = document.getElementById("uptime-chart-data");
const UptimeChartConstructor = window.Chart;

if (uptimeChartDataElement && typeof UptimeChartConstructor === "function") {
  if (Array.isArray(window.__gaviaUptimeCharts)) {
    for (const chart of window.__gaviaUptimeCharts) {
      chart.destroy();
    }
  }
  window.__gaviaUptimeCharts = [];

  let chartData = null;
  try {
    chartData = JSON.parse(uptimeChartDataElement.textContent || "{}");
  } catch {
    chartData = null;
  }

  if (chartData) {
    const hasValues = (series = []) =>
      Array.isArray(series) &&
      series.some((value) => value !== null && value !== undefined && !Number.isNaN(Number(value)));

    const showEmptyState = (canvas, message) => {
      if (!canvas) {
        return;
      }

      const frame = canvas.closest(".dashboard-chart-frame");
      if (frame) {
        frame.hidden = true;
      } else {
        canvas.hidden = true;
      }
      const note = document.createElement("p");
      note.className = "dashboard-chart-empty";
      note.textContent = message;
      (frame || canvas).after(note);
    };

    const resultsCanvas = document.getElementById("uptime-results-chart");
    if (resultsCanvas) {
      const labels = chartData.labels || [];
      const status = chartData.status || [];
      const latency = chartData.latency || [];

      if (!labels.length || (!hasValues(status) && !hasValues(latency))) {
        showEmptyState(resultsCanvas, "No recent uptime samples are available for this monitor.");
      } else {
        const chart = new UptimeChartConstructor(resultsCanvas, {
          type: "bar",
          data: {
            labels,
            datasets: [
              {
                type: "bar",
                label: "Availability",
                data: status,
                yAxisID: "y1",
                backgroundColor: status.map((value) => (Number(value) === 1 ? "#15803d" : "#b91c1c")),
                borderColor: "transparent",
              },
              {
                type: "line",
                label: "Latency ms",
                data: latency,
                yAxisID: "y",
                borderColor: "#1d4ed8",
                backgroundColor: "rgba(29, 78, 216, 0.18)",
                tension: 0.25,
                fill: false,
                spanGaps: true,
              },
            ],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            normalized: true,
            resizeDelay: 150,
            interaction: {
              mode: "index",
              intersect: false,
            },
            plugins: {
              legend: {
                position: "bottom",
              },
              tooltip: {
                callbacks: {
                  label(context) {
                    if (context.dataset.label === "Availability") {
                      return `Availability: ${Number(context.parsed.y) === 1 ? "Up" : "Down"}`;
                    }
                    return `Latency ms: ${Number(context.parsed.y || 0).toFixed(0)}`;
                  },
                },
              },
            },
            scales: {
              y: {
                title: {
                  display: true,
                  text: "Latency ms",
                },
              },
              y1: {
                min: 0,
                max: 1,
                position: "right",
                grid: {
                  drawOnChartArea: false,
                },
                ticks: {
                  callback(value) {
                    return Number(value) === 1 ? "Up" : "Down";
                  },
                },
              },
            },
          },
        });
        window.__gaviaUptimeCharts.push(chart);
      }
    }

    const distributionCanvas = document.getElementById("uptime-distribution-chart");
    if (distributionCanvas) {
      const counts = [chartData.up || 0, chartData.down || 0];
      if (!hasValues(counts)) {
        showEmptyState(distributionCanvas, "No recent status distribution is available yet.");
      } else {
        const chart = new UptimeChartConstructor(distributionCanvas, {
          type: "doughnut",
          data: {
            labels: ["Up", "Down"],
            datasets: [
              {
                data: counts,
                backgroundColor: ["#15803d", "#b91c1c"],
                borderColor: "transparent",
              },
            ],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            plugins: {
              legend: {
                position: "bottom",
              },
            },
          },
        });
        window.__gaviaUptimeCharts.push(chart);
      }
    }
  }
}

// @license-end

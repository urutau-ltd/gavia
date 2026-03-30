// @license magnet:?xt=urn:btih:0b31508aeb0634b347b8270c7bee4d411b5d4109&dn=agpl-3.0.txt AGPL-3.0-or-later
const uptimeChartDataElement = document.getElementById("uptime-chart-data");
const UptimeChartConstructor = window.Chart;

const resizeUptimeCharts = () => {
  if (!Array.isArray(window.__gaviaUptimeCharts)) {
    return;
  }

  window.requestAnimationFrame(() => {
    for (const chart of window.__gaviaUptimeCharts) {
      chart.resize();
    }
  });
};

if (!window.__gaviaUptimeTabsBound) {
  document.addEventListener("missing-change", (event) => {
    const tablist = event.target;
    if (!(tablist instanceof Element) || tablist.getAttribute("aria-label") !== "Uptime sections") {
      return;
    }

    resizeUptimeCharts();
  });
  window.__gaviaUptimeTabsBound = true;
}

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
      const availability = chartData.availability || [];

      if (!labels.length || !hasValues(availability)) {
        showEmptyState(resultsCanvas, "No recent uptime samples are available for this monitor.");
      } else {
        const chart = new UptimeChartConstructor(resultsCanvas, {
          type: "bar",
          data: {
            labels,
            datasets: [
              {
                label: "Availability",
                data: availability,
                backgroundColor: availability.map((value) => (Number(value) === 1 ? "#15803d" : "#b91c1c")),
                borderColor: "transparent",
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
                    return `Availability: ${Number(context.parsed.y) === 1 ? "Up" : "Down"}`;
                  },
                },
              },
            },
            scales: {
              y: {
                min: 0,
                max: 1,
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

    const latencyCanvas = document.getElementById("uptime-latency-chart");
    if (latencyCanvas) {
      const labels = chartData.labels || [];
      const latency = chartData.latency || [];
      if (!labels.length || !hasValues(latency)) {
        showEmptyState(latencyCanvas, "No latency samples are available for this monitor yet.");
      } else {
        const chart = new UptimeChartConstructor(latencyCanvas, {
          type: "line",
          data: {
            labels,
            datasets: [
              {
                label: "Latency ms",
                data: latency,
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
            plugins: {
              legend: {
                position: "bottom",
              },
            },
            scales: {
              y: {
                title: {
                  display: true,
                  text: "Latency ms",
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

    const statusCodeCanvas = document.getElementById("uptime-status-code-chart");
    if (statusCodeCanvas) {
      const labels = chartData.status_code_labels || [];
      const counts = chartData.status_code_counts || [];
      if (!labels.length || !hasValues(counts)) {
        showEmptyState(statusCodeCanvas, "No HTTP status codes have been recorded yet.");
      } else {
        const chart = new UptimeChartConstructor(statusCodeCanvas, {
          type: "bar",
          data: {
            labels,
            datasets: [
              {
                label: "Observed responses",
                data: counts,
                backgroundColor: "#0369a1",
                borderColor: "transparent",
              },
            ],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            normalized: true,
            resizeDelay: 150,
            plugins: {
              legend: {
                position: "bottom",
              },
            },
            scales: {
              y: {
                beginAtZero: true,
                ticks: {
                  precision: 0,
                },
              },
            },
          },
        });
        window.__gaviaUptimeCharts.push(chart);
      }
    }
  }
}

resizeUptimeCharts();

// @license-end

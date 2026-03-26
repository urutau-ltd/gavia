for (const element of document.querySelectorAll("[data-dashboard-relative-date]")) {
  const rawDate = element.getAttribute("datetime");
  const direction = element.getAttribute("data-dashboard-relative-date");
  if (!rawDate || !direction) {
    continue;
  }

  const current = new Date();
  const target = new Date(`${rawDate}T00:00:00`);
  if (Number.isNaN(target.getTime())) {
    continue;
  }

  const today = new Date(current.getFullYear(), current.getMonth(), current.getDate());
  const dayMS = 24 * 60 * 60 * 1000;
  const diffDays = Math.round((target.getTime() - today.getTime()) / dayMS);

  let suffix = "";
  if (direction === "future") {
    if (diffDays === 0) {
      suffix = "today";
    } else if (diffDays === 1) {
      suffix = "in 1 day";
    } else if (diffDays > 1) {
      suffix = `in ${diffDays} days`;
    }
  }

  if (direction === "past") {
    if (diffDays === 0) {
      suffix = "today";
    } else if (diffDays === -1) {
      suffix = "1 day ago";
    } else if (diffDays < -1) {
      suffix = `${Math.abs(diffDays)} days ago`;
    }
  }

  if (!suffix) {
    continue;
  }

  element.textContent = `${element.textContent} (${suffix})`;
}

const chartDataElement = document.getElementById("dashboard-chart-data");
const ChartConstructor = window.Chart;

if (chartDataElement && typeof ChartConstructor === "function") {
  if (Array.isArray(window.__gaviaDashboardCharts)) {
    for (const chart of window.__gaviaDashboardCharts) {
      chart.destroy();
    }
  }
  window.__gaviaDashboardCharts = [];

  let chartData = null;
  try {
    chartData = JSON.parse(chartDataElement.textContent || "{}");
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

    const baseLineOptions = {
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
      },
    };

    const buildLineChart = (canvasID, config, emptyMessage) => {
      const canvas = document.getElementById(canvasID);
      if (!canvas) {
        return;
      }

      if (!Array.isArray(config.data.labels) || config.data.labels.length === 0) {
        showEmptyState(canvas, emptyMessage);
        return;
      }

      const validDataset = config.data.datasets.some((dataset) => hasValues(dataset.data));
      if (!validDataset) {
        showEmptyState(canvas, emptyMessage);
        return;
      }

      const chart = new ChartConstructor(canvas, config);
      window.__gaviaDashboardCharts.push(chart);
    };

    buildLineChart(
      "expense-history-chart",
      {
        type: "line",
        data: {
          labels: chartData.expense_history?.labels || [],
          datasets: [
            {
              label: "MXN",
              data: chartData.expense_history?.mxn || [],
              borderColor: "#d97706",
              backgroundColor: "rgba(217, 119, 6, 0.18)",
              tension: 0.25,
              fill: false,
            },
            {
              label: "USD",
              data: chartData.expense_history?.usd || [],
              borderColor: "#0f766e",
              backgroundColor: "rgba(15, 118, 110, 0.18)",
              tension: 0.25,
              fill: false,
            },
            {
              label: "XMR",
              data: chartData.expense_history?.xmr || [],
              borderColor: "#5b21b6",
              backgroundColor: "rgba(91, 33, 182, 0.18)",
              tension: 0.25,
              fill: false,
            },
          ],
        },
        options: {
          ...baseLineOptions,
          plugins: {
            ...baseLineOptions.plugins,
            tooltip: {
              callbacks: {
                label(context) {
                  const value = Number(context.parsed.y || 0);
                  const digits = context.dataset.label === "XMR" ? 6 : 2;
                  return `${context.dataset.label}: ${value.toFixed(digits)}`;
                },
              },
            },
          },
        },
      },
      "No expense history is available yet.",
    );

    buildLineChart(
      "fx-history-chart",
      {
        type: "line",
        data: {
          labels: chartData.fx_history?.labels || [],
          datasets: [
            {
              label: "MXN to USD",
              data: chartData.fx_history?.mxn_to_usd || [],
              borderColor: "#0f766e",
              backgroundColor: "rgba(15, 118, 110, 0.18)",
              tension: 0.25,
              fill: false,
              yAxisID: "y",
            },
            {
              label: "MXN to XMR",
              data: chartData.fx_history?.mxn_to_xmr || [],
              borderColor: "#5b21b6",
              backgroundColor: "rgba(91, 33, 182, 0.18)",
              tension: 0.25,
              fill: false,
              yAxisID: "y1",
            },
          ],
        },
        options: {
          ...baseLineOptions,
          scales: {
            y: {
              title: {
                display: true,
                text: "USD",
              },
            },
            y1: {
              position: "right",
              grid: {
                drawOnChartArea: false,
              },
              title: {
                display: true,
                text: "XMR",
              },
            },
          },
          plugins: {
            ...baseLineOptions.plugins,
            tooltip: {
              callbacks: {
                label(context) {
                  const value = Number(context.parsed.y || 0);
                  const digits = context.dataset.yAxisID === "y1" ? 8 : 6;
                  return `${context.dataset.label}: ${value.toFixed(digits)}`;
                },
              },
            },
          },
        },
      },
      "No FX samples have been stored yet.",
    );

    buildLineChart(
      "runtime-history-chart",
      {
        type: "line",
        data: {
          labels: chartData.runtime_history?.labels || [],
          datasets: [
            {
              label: "Heap alloc MB",
              data: chartData.runtime_history?.heap_alloc_mb || [],
              borderColor: "#1d4ed8",
              backgroundColor: "rgba(29, 78, 216, 0.18)",
              tension: 0.25,
              fill: true,
              yAxisID: "y",
            },
            {
              label: "Goroutines",
              data: chartData.runtime_history?.goroutines || [],
              borderColor: "#b91c1c",
              backgroundColor: "rgba(185, 28, 28, 0.18)",
              tension: 0.25,
              fill: false,
              yAxisID: "y1",
            },
            {
              label: "DB open connections",
              data: chartData.runtime_history?.db_open_connections || [],
              borderColor: "#7c3aed",
              backgroundColor: "rgba(124, 58, 237, 0.18)",
              tension: 0.25,
              fill: false,
              yAxisID: "y1",
            },
          ],
        },
        options: {
          ...baseLineOptions,
          scales: {
            y: {
              title: {
                display: true,
                text: "MB",
              },
            },
            y1: {
              position: "right",
              grid: {
                drawOnChartArea: false,
              },
              title: {
                display: true,
                text: "Count",
              },
            },
          },
        },
      },
      "No runtime diagnostics have been sampled yet.",
    );

    const uptimeCanvas = document.getElementById("uptime-status-chart");
    if (uptimeCanvas) {
      const counts = chartData.uptime_status?.counts || [];
      if (hasValues(counts)) {
        const chart = new ChartConstructor(uptimeCanvas, {
          type: "doughnut",
          data: {
            labels: chartData.uptime_status?.labels || [],
            datasets: [
              {
                data: counts,
                backgroundColor: ["#15803d", "#b91c1c", "#a16207", "#475569"],
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
        window.__gaviaDashboardCharts.push(chart);
      } else {
        showEmptyState(uptimeCanvas, "No uptime monitor states are available yet.");
      }
    }
  }
}

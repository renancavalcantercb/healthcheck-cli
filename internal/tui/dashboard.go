package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// Styles
var (
	// Colors
	primaryColor    = lipgloss.Color("#00D9FF")
	successColor    = lipgloss.Color("#00FF87")
	warningColor    = lipgloss.Color("#FFFF00")
	errorColor      = lipgloss.Color("#FF5F87")
	mutedColor      = lipgloss.Color("#626262")
	backgroundColor = lipgloss.Color("#1a1a1a")

	// Base styles
	baseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(backgroundColor)

	// Header style
	headerStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	// Table styles
	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(mutedColor).
				Padding(0, 1)

	tableRowStyle = lipgloss.NewStyle().
			Padding(0, 2)

	// Status styles
	statusUpStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	statusDownStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	statusSlowStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(1, 2).
			Margin(0, 1, 1, 0)

	// Footer style
	footerStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(mutedColor).
			Padding(1, 0, 0, 0).
			Margin(1, 0, 0, 0)
)

type Model struct {
	results       []types.Result
	lastUpdate    time.Time
	width         int
	height        int
	sortBy        string
	filterBy      string
	showHelp      bool
	refreshRate   time.Duration
	stats         Stats
	history       map[string][]types.Result
	memoryConfig  types.MemoryManagementConfig
	lastCleanup   time.Time
}

type Stats struct {
	TotalChecks   int
	UpCount       int
	DownCount     int
	SlowCount     int
	AvgResponse   time.Duration
	UptimePercent float64
}

type TickMsg time.Time

func New() Model {
	// Default memory configuration if none provided
	defaultMemConfig := types.MemoryManagementConfig{
		Enabled:                true,
		MaxHistoryPerService:   100,
		MaxHistoryAge:          24 * time.Hour,
		CleanupInterval:        5 * time.Minute,
		MaxTotalMemoryMB:       100,
	}
	
	return NewWithConfig(defaultMemConfig)
}

func NewWithConfig(memConfig types.MemoryManagementConfig) Model {
	return Model{
		results:       make([]types.Result, 0),
		lastUpdate:    time.Now(),
		sortBy:        "name",
		filterBy:      "all",
		refreshRate:   5 * time.Second,
		history:       make(map[string][]types.Result),
		memoryConfig:  memConfig,
		lastCleanup:   time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "r":
			// Manual refresh
			m.lastUpdate = time.Now()
			return m, tickCmd()

		case "h":
			m.showHelp = !m.showHelp
			return m, nil

		case "s":
			// Sort by response time
			if m.sortBy == "response_time" {
				m.sortBy = "name"
			} else {
				m.sortBy = "response_time"
			}
			return m, nil

		case "f":
			// Filter by status
			switch m.filterBy {
			case "all":
				m.filterBy = "up"
			case "up":
				m.filterBy = "down"
			case "down":
				m.filterBy = "slow"
			case "slow":
				m.filterBy = "all"
			}
			return m, nil
		}

	case TickMsg:
		m.lastUpdate = time.Now()
		return m, tickCmd()

	case []types.Result:
		m.updateResults(msg)
		return m, nil
	}

	return m, nil
}

func (m *Model) updateResults(results []types.Result) {
	m.results = results
	m.calculateStats()

	// Update history for each result with memory management
	for _, result := range results {
		if _, exists := m.history[result.Name]; !exists {
			initialCapacity := m.memoryConfig.MaxHistoryPerService
			if initialCapacity <= 0 {
				initialCapacity = 100
			}
			m.history[result.Name] = make([]types.Result, 0, initialCapacity)
		}

		// Add new result and manage history size
		m.addToHistory(result.Name, result)
	}

	// Perform cleanup if needed
	m.performCleanupIfNeeded()
}

func (m *Model) addToHistory(serviceName string, result types.Result) {
	if !m.memoryConfig.Enabled {
		// If memory management is disabled, use simple approach
		history := m.history[serviceName]
		if len(history) >= 100 {
			history = history[1:]
		}
		m.history[serviceName] = append(history, result)
		return
	}

	history := m.history[serviceName]
	maxSize := m.memoryConfig.MaxHistoryPerService
	if maxSize <= 0 {
		maxSize = 100
	}

	// Remove oldest entries if we exceed the limit
	if len(history) >= maxSize {
		// Remove oldest entries
		removeCount := len(history) - maxSize + 1
		history = history[removeCount:]
	}

	// Add new result
	m.history[serviceName] = append(history, result)
}

func (m *Model) performCleanupIfNeeded() {
	if !m.memoryConfig.Enabled {
		return
	}

	now := time.Now()
	
	// Check if it's time for cleanup
	if now.Sub(m.lastCleanup) < m.memoryConfig.CleanupInterval {
		return
	}

	m.performMemoryCleanup()
	m.lastCleanup = now
}

func (m *Model) performMemoryCleanup() {
	if !m.memoryConfig.Enabled {
		return
	}

	now := time.Now()
	maxAge := m.memoryConfig.MaxHistoryAge
	
	// Clean up old entries based on age
	for serviceName, history := range m.history {
		if len(history) == 0 {
			continue
		}

		// Find the first entry that's not too old
		firstValidIndex := 0
		for i, result := range history {
			if maxAge > 0 && now.Sub(result.Timestamp) > maxAge {
				firstValidIndex = i + 1
			} else {
				break
			}
		}

		// Remove old entries if any were found
		if firstValidIndex > 0 {
			if firstValidIndex >= len(history) {
				// All entries are too old, keep empty slice
				m.history[serviceName] = make([]types.Result, 0, m.memoryConfig.MaxHistoryPerService)
			} else {
				// Keep only the recent entries
				m.history[serviceName] = history[firstValidIndex:]
			}
		}
	}

	// Remove empty service histories to free memory
	for serviceName, history := range m.history {
		if len(history) == 0 {
			delete(m.history, serviceName)
		}
	}
}

// GetMemoryStats returns current memory usage statistics
func (m *Model) GetMemoryStats() map[string]interface{} {
	totalEntries := 0
	serviceCount := len(m.history)
	
	oldestEntry := time.Now()
	newestEntry := time.Time{}
	
	for _, history := range m.history {
		totalEntries += len(history)
		for _, result := range history {
			if result.Timestamp.Before(oldestEntry) {
				oldestEntry = result.Timestamp
			}
			if result.Timestamp.After(newestEntry) {
				newestEntry = result.Timestamp
			}
		}
	}

	return map[string]interface{}{
		"total_entries":   totalEntries,
		"service_count":   serviceCount,
		"oldest_entry":    oldestEntry,
		"newest_entry":    newestEntry,
		"last_cleanup":    m.lastCleanup,
		"memory_enabled":  m.memoryConfig.Enabled,
		"max_per_service": m.memoryConfig.MaxHistoryPerService,
		"max_age":         m.memoryConfig.MaxHistoryAge,
	}
}

func (m *Model) calculateStats() {
	m.stats = Stats{
		TotalChecks: len(m.results),
	}

	if len(m.results) == 0 {
		return
	}

	var totalResponseTime time.Duration
	upCount := 0

	for _, result := range m.results {
		switch result.Status {
		case types.StatusUp:
			m.stats.UpCount++
			upCount++
		case types.StatusDown:
			m.stats.DownCount++
		case types.StatusSlow:
			m.stats.SlowCount++
			upCount++ // Slow is still "up"
		}
		totalResponseTime += result.ResponseTime
	}

	m.stats.AvgResponse = totalResponseTime / time.Duration(len(m.results))
	m.stats.UptimePercent = float64(upCount) / float64(len(m.results)) * 100
}

func (m Model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	header := m.renderHeader()
	httpTable := m.renderHTTPTable()
	tcpTable := m.renderTCPTable()
	metrics := m.renderMetrics()
	footer := m.renderFooter()

	// Layout based on terminal size
	if m.width < 130 {
		// Narrow layout - stack vertically
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			httpTable,
			tcpTable,
			metrics,
			footer,
		)
	} else {
		// Wide layout - side by side with more space
		tables := lipgloss.JoinVertical(lipgloss.Left, httpTable, tcpTable)

		// Add some spacing between tables and metrics
		tablesWithSpacing := lipgloss.NewStyle().MarginRight(2).Render(tables)
		metricsWithSpacing := lipgloss.NewStyle().MarginLeft(1).Render(metrics)

		body := lipgloss.JoinHorizontal(lipgloss.Top, tablesWithSpacing, metricsWithSpacing)
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			body,
			footer,
		)
	}
}

func (m Model) renderHeader() string {
	title := "⚡ HealthCheck Dashboard"
	timestamp := m.lastUpdate.Format("2006-01-02 15:04:05")

	statusSummary := fmt.Sprintf("🟢 %d UP  🟡 %d SLOW  🔴 %d DOWN",
		m.stats.UpCount, m.stats.SlowCount, m.stats.DownCount)

	uptime := fmt.Sprintf("📊 %.1f%% Uptime", m.stats.UptimePercent)
	avgResponse := fmt.Sprintf("⚡ %v Avg", m.stats.AvgResponse.Truncate(time.Millisecond))

	headerLeft := fmt.Sprintf("%s  •  %s", title, statusSummary)
	headerRight := fmt.Sprintf("%s  •  %s  •  %s", uptime, avgResponse, timestamp)

	// Calculate spacing
	totalWidth := m.width - 4 // Account for border padding
	spacing := totalWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if spacing < 0 {
		spacing = 0
	}

	headerContent := headerLeft + strings.Repeat(" ", spacing) + headerRight

	return headerStyle.Width(m.width - 2).Render(headerContent)
}

func (m Model) renderHTTPTable() string {
	httpResults := m.filterResults("http")
	if len(httpResults) == 0 {
		return boxStyle.Render("🌐 No HTTP services configured")
	}

	m.sortResults(httpResults)

	var rows []string
	header := tableHeaderStyle.Render(
		fmt.Sprintf("%-22s %-42s %-14s %-16s", "NAME", "URL", "STATUS", "RESPONSE TIME"))
	rows = append(rows, header)

	for _, result := range httpResults {
		name := truncate(result.Name, 20)
		url := truncate(result.URL, 40)
		status := m.formatStatus(result.Status)
		responseTime := result.ResponseTime.Truncate(time.Millisecond).String()

		row := tableRowStyle.Render(
			fmt.Sprintf("%-22s %-42s %-14s %-16s", name, url, status, responseTime))
		rows = append(rows, row)
	}

	table := strings.Join(rows, "\n")
	title := fmt.Sprintf("🌐 HTTP Services (%d)", len(httpResults))

	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(title),
		"",
		table,
	))
}

func (m Model) renderTCPTable() string {
	tcpResults := m.filterResults("tcp")
	if len(tcpResults) == 0 {
		return boxStyle.Render("🔌 No TCP services configured")
	}

	m.sortResults(tcpResults)

	var rows []string
	header := tableHeaderStyle.Render(
		fmt.Sprintf("%-22s %-28s %-14s %-16s", "NAME", "HOST:PORT", "STATUS", "LATENCY"))
	rows = append(rows, header)

	for _, result := range tcpResults {
		name := truncate(result.Name, 20)
		host := truncate(result.URL, 26)
		status := m.formatStatus(result.Status)
		latency := result.ResponseTime.Truncate(time.Millisecond).String()

		row := tableRowStyle.Render(
			fmt.Sprintf("%-22s %-28s %-14s %-16s", name, host, status, latency))
		rows = append(rows, row)
	}

	table := strings.Join(rows, "\n")
	title := fmt.Sprintf("🔌 TCP Services (%d)", len(tcpResults))

	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(title),
		"",
		table,
	))
}

func (m Model) renderMetrics() string {
	uptime := fmt.Sprintf("%.1f%%", m.stats.UptimePercent)
	avgResponse := m.stats.AvgResponse.Truncate(time.Millisecond).String()
	totalChecks := fmt.Sprintf("%d", m.stats.TotalChecks)

	metrics := []string{
		fmt.Sprintf("📈 Uptime:       %s", uptime),
		fmt.Sprintf("⚡ Avg RT:       %s", avgResponse),
		fmt.Sprintf("📊 Checks:      %s", totalChecks),
		fmt.Sprintf("🔄 Updated:     %s", m.lastUpdate.Format("15:04:05")),
	}

	// Recent alerts/events
	alerts := []string{
		"📢 Recent Events:",
		"",
	}

	// Add some recent status changes
	hasIssues := false
	for _, result := range m.results {
		if !result.IsHealthy() {
			hasIssues = true
			icon := "🔴"
			if result.Status == types.StatusSlow {
				icon = "🟡"
			}
			alerts = append(alerts, fmt.Sprintf("%s %s", icon, truncate(result.Name, 16)))
		}
	}

	if !hasIssues {
		alerts = append(alerts, "✅ All services healthy")
	}

	metricsSection := strings.Join(metrics, "\n")
	alertsSection := strings.Join(alerts, "\n")

	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("📊 Metrics"),
		"",
		metricsSection,
		"",
		alertsSection,
	))
}

func (m Model) renderFooter() string {
	shortcuts := []string{
		"[q] quit",
		"[r] refresh",
		"[s] sort",
		"[f] filter",
		"[h] help",
	}

	filterInfo := fmt.Sprintf("Filter: %s", m.filterBy)
	sortInfo := fmt.Sprintf("Sort: %s", m.sortBy)
	refreshInfo := fmt.Sprintf("Auto-refresh: %v", m.refreshRate)

	left := strings.Join(shortcuts, "  ")
	right := fmt.Sprintf("%s  •  %s  •  %s", filterInfo, sortInfo, refreshInfo)

	// Calculate spacing
	totalWidth := m.width - 2
	spacing := totalWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if spacing < 0 {
		spacing = 0
	}

	content := left + strings.Repeat(" ", spacing) + right

	return footerStyle.Width(m.width).Render(content)
}

func (m Model) renderHelp() string {
	help := `
⚡ HealthCheck Dashboard - Help

KEYBOARD SHORTCUTS:
  q, Ctrl+C    Quit the application
  r            Manual refresh
  s            Toggle sort (name ↔ response time)
  f            Cycle filter (all → up → down → slow → all)
  h            Toggle this help screen

DASHBOARD SECTIONS:
  🌐 HTTP Services    Web endpoints and APIs
  🔌 TCP Services     Port connectivity checks
  📊 Metrics          Statistics and recent events

STATUS INDICATORS:
  🟢 UP      Service is responding correctly
  🟡 SLOW    Service is responding but slowly
  🔴 DOWN    Service is not responding

Press 'h' again to return to the dashboard.
`

	return baseStyle.
		Width(m.width).
		Height(m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(help)
}

func (m Model) formatStatus(status types.Status) string {
	switch status {
	case types.StatusUp:
		return statusUpStyle.Render("🟢 UP")
	case types.StatusDown:
		return statusDownStyle.Render("🔴 DOWN")
	case types.StatusSlow:
		return statusSlowStyle.Render("🟡 SLOW")
	default:
		return "❓ UNKNOWN"
	}
}

func (m Model) filterResults(serviceType string) []types.Result {
	var filtered []types.Result

	for _, result := range m.results {
		// Filter by service type
		if serviceType == "http" && !strings.HasPrefix(result.URL, "http") {
			continue
		}
		if serviceType == "tcp" && strings.HasPrefix(result.URL, "http") {
			continue
		}

		// Filter by status
		switch m.filterBy {
		case "up":
			if result.Status != types.StatusUp {
				continue
			}
		case "down":
			if result.Status != types.StatusDown {
				continue
			}
		case "slow":
			if result.Status != types.StatusSlow {
				continue
			}
		}

		filtered = append(filtered, result)
	}

	return filtered
}

func (m Model) sortResults(results []types.Result) {
	switch m.sortBy {
	case "response_time":
		sort.Slice(results, func(i, j int) bool {
			return results[i].ResponseTime < results[j].ResponseTime
		})
	default: // "name"
		sort.Slice(results, func(i, j int) bool {
			return results[i].Name < results[j].Name
		})
	}
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// UpdateResults is a helper to send results to the model
func UpdateResults(results []types.Result) tea.Cmd {
	return func() tea.Msg {
		return results
	}
}

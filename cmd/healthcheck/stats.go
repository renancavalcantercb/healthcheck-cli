package main

import (
	"fmt"
	"strings"
	"time"
)

// ShowStats displays statistics from stored data
func (a *App) ShowStats(serviceName, sinceStr string, jsonOutput bool) error {
	if a.storage == nil {
		return fmt.Errorf("storage not available - stats require data persistence")
	}

	since := time.Now().Add(-24 * time.Hour)
	if sinceStr != "" {
		duration, err := time.ParseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid duration format: %s (use: 1h, 24h, 7d, etc.)", sinceStr)
		}
		since = time.Now().Add(-duration)
	}

	if serviceName != "" {
		return a.showServiceStats(serviceName, since, jsonOutput)
	} else {
		return a.showAllStats(since, jsonOutput)
	}
}

// ShowHistory displays historical data for a service
func (a *App) ShowHistory(serviceName string, limit int, sinceStr string) error {
	if a.storage == nil {
		return fmt.Errorf("storage not available - history requires data persistence")
	}

	since := time.Now().Add(-24 * time.Hour)
	if sinceStr != "" {
		duration, err := time.ParseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid duration format: %s", sinceStr)
		}
		since = time.Now().Add(-duration)
	}

	history, err := a.storage.GetServiceHistory(serviceName, since, limit)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	if len(history) == 0 {
		fmt.Printf("📊 No history found for service '%s'\n", serviceName)
		return nil
	}

	fmt.Printf("📈 History for %s (last %d checks)\n", serviceName, len(history))
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("%-19s %-8s %-12s %-30s\n", "TIMESTAMP", "STATUS", "RESPONSE", "ERROR")
	fmt.Printf("───────────────────────────────────────────────────────────────\n")

	for _, record := range history {
		timestamp := record.Timestamp.Format("01-02 15:04:05")
		
		var status string
		switch record.Status {
		case 0: // StatusUp
			status = "🟢 UP"
		case 1: // StatusDown
			status = "🔴 DOWN"
		case 2: // StatusSlow
			status = "🟡 SLOW"
		default:
			status = "❓ UNK"
		}

		response := fmt.Sprintf("%dms", record.ResponseTimeMs)
		if record.StatusCode > 0 {
			response += fmt.Sprintf(" (%d)", record.StatusCode)
		}

		errorMsg := truncateString(record.Error, 28)

		fmt.Printf("%-19s %-8s %-12s %-30s\n", timestamp, status, response, errorMsg)
	}

	return nil
}

// ShowDatabaseInfo displays information about the database
func (a *App) ShowDatabaseInfo() error {
	if a.storage == nil {
		return fmt.Errorf("storage not available")
	}

	info, err := a.storage.GetDatabaseInfo()
	if err != nil {
		return fmt.Errorf("failed to get database info: %w", err)
	}

	fmt.Println("🗄️  Database Information")
	fmt.Println("═══════════════════════════════════")
	
	if path, ok := info["database_path"].(string); ok {
		fmt.Printf("📁 Path:            %s\n", path)
	}
	
	if totalRecords, ok := info["total_records"].(int64); ok {
		fmt.Printf("📊 Total Records:   %d\n", totalRecords)
	}
	
	if totalServices, ok := info["total_services"].(int64); ok {
		fmt.Printf("🏷️  Services:        %d\n", totalServices)
	}
	
	if sizeBytes, ok := info["database_size_bytes"].(int64); ok {
		sizeKB := float64(sizeBytes) / 1024
		sizeMB := sizeKB / 1024
		if sizeMB > 1 {
			fmt.Printf("💾 Size:            %.1f MB\n", sizeMB)
		} else {
			fmt.Printf("💾 Size:            %.1f KB\n", sizeKB)
		}
	}
	
	if oldest, ok := info["oldest_record"].(time.Time); ok {
		fmt.Printf("📅 Oldest Record:   %s\n", oldest.Format("2006-01-02 15:04:05"))
	}
	
	if newest, ok := info["newest_record"].(time.Time); ok {
		fmt.Printf("🕐 Newest Record:   %s\n", newest.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// showServiceStats shows detailed stats for a specific service
func (a *App) showServiceStats(serviceName string, since time.Time, jsonOutput bool) error {
	stats, err := a.storage.GetServiceStats(serviceName, since)
	if err != nil {
		return fmt.Errorf("failed to get stats for %s: %w", serviceName, err)
	}

	if jsonOutput {
		fmt.Printf("{\"service\":\"%s\",\"stats\":%+v}\n", serviceName, stats)
		return nil
	}

	fmt.Printf("📊 Statistics for %s\n", stats.Name)
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("🔗 URL:              %s\n", stats.URL)
	fmt.Printf("📝 Type:             %s\n", strings.ToUpper(stats.CheckType))
	fmt.Printf("📈 Uptime:           %.2f%%\n", stats.UptimePercent)
	fmt.Printf("✅ Successful:       %d\n", stats.SuccessfulChecks)
	fmt.Printf("❌ Failed:           %d\n", stats.FailedChecks)
	fmt.Printf("📊 Total Checks:     %d\n", stats.TotalChecks)
	fmt.Printf("⚡ Avg Response:     %.0fms\n", stats.AvgResponseTimeMs)
	fmt.Printf("🚀 Min Response:     %dms\n", stats.MinResponseTimeMs)
	fmt.Printf("🐌 Max Response:     %dms\n", stats.MaxResponseTimeMs)
	fmt.Printf("🕐 Last Check:       %s\n", stats.LastCheck.Format("2006-01-02 15:04:05"))

	if !stats.LastSuccess.IsZero() {
		fmt.Printf("✅ Last Success:     %s\n", stats.LastSuccess.Format("2006-01-02 15:04:05"))
	}
	if !stats.LastFailure.IsZero() {
		fmt.Printf("❌ Last Failure:     %s\n", stats.LastFailure.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// showAllStats shows stats for all services
func (a *App) showAllStats(since time.Time, jsonOutput bool) error {
	allStats, err := a.storage.GetAllServiceStats(since)
	if err != nil {
		return fmt.Errorf("failed to get all stats: %w", err)
	}

	if len(allStats) == 0 {
		fmt.Println("📊 No statistics available yet")
		fmt.Println("💡 Run some checks first to generate stats")
		return nil
	}

	if jsonOutput {
		fmt.Printf("{\"services\":%+v}\n", allStats)
		return nil
	}

	fmt.Printf("📊 Service Statistics (since %s)\n", since.Format("2006-01-02 15:04"))
	fmt.Printf("═══════════════════════════════════════════════════════════════════════\n")
	fmt.Printf("%-20s %-12s %-8s %-10s %-12s %-15s\n", 
		"SERVICE", "TYPE", "UPTIME", "CHECKS", "AVG RT", "LAST CHECK")
	fmt.Printf("───────────────────────────────────────────────────────────────────────\n")

	for _, stats := range allStats {
		name := truncateString(stats.Name, 18)
		checkType := strings.ToUpper(stats.CheckType)
		uptime := fmt.Sprintf("%.1f%%", stats.UptimePercent)
		checks := fmt.Sprintf("%d", stats.TotalChecks)
		avgRT := fmt.Sprintf("%.0fms", stats.AvgResponseTimeMs)
		lastCheck := stats.LastCheck.Format("15:04:05")

		uptimeColor := ""
		if stats.UptimePercent >= 99.0 {
			uptimeColor = "🟢"
		} else if stats.UptimePercent >= 95.0 {
			uptimeColor = "🟡"
		} else {
			uptimeColor = "🔴"
		}

		fmt.Printf("%-20s %-12s %s%-7s %-10s %-12s %-15s\n", 
			name, checkType, uptimeColor, uptime, checks, avgRT, lastCheck)
	}

	return nil
}

// truncateString truncates a string to a specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/interfaces"
)

// ShowHistory displays historical data for a service (CLI wrapper)
func ShowHistory(app interfaces.Application, serviceName string, limit int, sinceStr string) error {
	since := time.Now().Add(-24 * time.Hour)
	if sinceStr != "" {
		duration, err := time.ParseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid duration format: %s", sinceStr)
		}
		since = time.Now().Add(-duration)
	}

	history, err := app.Stats().GetHistory(serviceName, since, limit)
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

// ShowDatabaseInfo displays information about the database (CLI wrapper)
func ShowDatabaseInfo(app interfaces.Application) error {
	info, err := app.Stats().GetDatabaseInfo()
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

// ValidateConfig validates a configuration file (CLI wrapper)
func ValidateConfig(configFile string) error {
	fmt.Printf("🔍 Validating configuration file: %s\n", configFile)
	
	_, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
		return err
	}
	
	fmt.Println("✅ Configuration is valid!")
	return nil
}

// GenerateExampleConfig generates an example configuration file (CLI wrapper)
func GenerateExampleConfig(outputFile string) error {
	if outputFile == "" {
		return config.SaveExample("")
	}
	
	if err := config.SaveExample(outputFile); err != nil {
		return fmt.Errorf("failed to generate example config: %w", err)
	}
	
	fmt.Printf("✅ Example configuration saved to %s\n", outputFile)
	return nil
}

// ShowStats displays statistics from stored data (CLI wrapper)
func ShowStats(app interfaces.Application, serviceName, sinceStr string, jsonOutput bool) error {
	since := time.Now().Add(-24 * time.Hour)
	if sinceStr != "" {
		duration, err := time.ParseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid duration format: %s (use: 1h, 24h, 7d, etc.)", sinceStr)
		}
		since = time.Now().Add(-duration)
	}

	if serviceName != "" {
		return showServiceStats(app, serviceName, since, jsonOutput)
	} else {
		return showAllStats(app, since, jsonOutput)
	}
}

// showServiceStats shows detailed stats for a specific service
func showServiceStats(app interfaces.Application, serviceName string, since time.Time, jsonOutput bool) error {
	stats, err := app.Stats().GetServiceStats(serviceName, since)
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
func showAllStats(app interfaces.Application, since time.Time, jsonOutput bool) error {
	allStats, err := app.Stats().GetAllStats(since)
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
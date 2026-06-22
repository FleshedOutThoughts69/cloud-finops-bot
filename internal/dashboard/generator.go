// internal/dashboard/generator.go

package dashboard

import (
    "encoding/json"
    "fmt"
    "time"

    "cloud-finops-bot/internal/dynamodb"
)

// DashboardData represents the data structure embedded in the HTML.
type DashboardData struct {
    TotalSavings   float64                 `json:"totalSavings"`
    LastUpdated    string                  `json:"lastUpdated"`
    HealthStatus   string                  `json:"healthStatus"`
    Breakdown      ResourceBreakdown       `json:"breakdown"`
    MonthlyTrend   []MonthlyTrendData      `json:"monthlyTrend"`
    RecentActivity []dynamodb.ResourceState `json:"recentActivity"`
}

// ResourceBreakdown aggregates savings by resource type.
type ResourceBreakdown struct {
    EBS      ResourceMetrics `json:"ebs"`
    EIP      ResourceMetrics `json:"eip"`
    Snapshot ResourceMetrics `json:"snapshot"`
    RDS      ResourceMetrics `json:"rds"`
}

// ResourceMetrics holds count and savings for a resource type.
type ResourceMetrics struct {
    Savings float64 `json:"savings"`
    Count   int     `json:"count"`
}

// MonthlyTrendData represents savings per month.
type MonthlyTrendData struct {
    Month   string  `json:"month"`
    Savings float64 `json:"savings"`
}

// GenerateDashboardHTML creates the full HTML page with embedded data.
func GenerateDashboardHTML(records []dynamodb.ResourceState, totalSavings float64, lastUpdated time.Time, healthStatus string) string {
    data := DashboardData{
        TotalSavings: totalSavings,
        LastUpdated:  lastUpdated.UTC().Format("2006-01-02 15:04:05 UTC"),
        HealthStatus: healthStatus,
        Breakdown:    calculateBreakdown(records),
        MonthlyTrend: calculateMonthlyTrend(records),
        RecentActivity: getRecentActivity(records, 20),
    }

    jsonData, _ := json.Marshal(data)

    return buildHTML(string(jsonData))
}

// calculateBreakdown aggregates savings by resource type.
func calculateBreakdown(records []dynamodb.ResourceState) ResourceBreakdown {
    breakdown := ResourceBreakdown{}
    for _, r := range records {
        if r.ActionTaken != "DELETED" && r.ActionTaken != "STOPPED" {
            continue
        }
        savings := 0.0
        if r.EstimatedSavings != nil {
            savings = *r.EstimatedSavings
        }
        switch r.ResourceType {
        case "EBS_VOLUME":
            breakdown.EBS.Savings += savings
            breakdown.EBS.Count++
        case "EIP":
            breakdown.EIP.Savings += savings
            breakdown.EIP.Count++
        case "SNAPSHOT":
            breakdown.Snapshot.Savings += savings
            breakdown.Snapshot.Count++
        case "RDS_INSTANCE":
            breakdown.RDS.Savings += savings
            breakdown.RDS.Count++
        }
    }
    return breakdown
}

// calculateMonthlyTrend groups savings by month.
func calculateMonthlyTrend(records []dynamodb.ResourceState) []MonthlyTrendData {
    monthlyMap := make(map[string]float64)
    for _, r := range records {
        if r.ActionTaken != "DELETED" && r.ActionTaken != "STOPPED" {
            continue
        }
        if r.EstimatedSavings == nil || r.DeletionTimestamp == nil {
            continue
        }
        t := time.Unix(*r.DeletionTimestamp, 0)
        monthKey := t.Format("Jan")
        monthlyMap[monthKey] += *r.EstimatedSavings
    }

    // Convert to sorted slice
    monthOrder := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
    var trend []MonthlyTrendData
    for _, month := range monthOrder {
        if val, ok := monthlyMap[month]; ok {
            trend = append(trend, MonthlyTrendData{Month: month, Savings: val})
        }
    }
    return trend
}

// getRecentActivity returns the most recent records up to limit.
func getRecentActivity(records []dynamodb.ResourceState, limit int) []dynamodb.ResourceState {
    if len(records) > limit {
        return records[:limit]
    }
    return records
}

// buildHTML returns the complete HTML document.
func buildHTML(jsonData string) string {
    return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cloud FinOps Bot Dashboard</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700;800&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background: #0f1419;
            color: #e2e8f0;
            padding: 20px;
            min-height: 100vh;
        }
        .dashboard { max-width: 1200px; margin: 0 auto; }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 16px 24px;
            background: #1a202c;
            border-radius: 12px;
            border: 1px solid #2d3748;
            margin-bottom: 24px;
            flex-wrap: wrap;
            gap: 8px;
        }
        .header-left { display: flex; align-items: center; gap: 12px; }
        .header-left h1 { font-size: 20px; font-weight: 700; color: #fff; }
        .header-left h1 i { color: #4299e1; margin-right: 8px; }
        .badge {
            background: #2d3748; padding: 2px 10px; border-radius: 12px;
            font-size: 11px; font-weight: 600; color: #a0aec0;
        }
        .header-center { display: flex; align-items: center; }
        .health-indicator {
            display: flex; align-items: center; gap: 8px;
            font-size: 14px; font-weight: 600; padding: 4px 12px; border-radius: 20px;
        }
        .health-indicator i { font-size: 10px; }
        .health-indicator.healthy { color: #48bb78; background: rgba(72,187,120,0.15); }
        .health-indicator.warning { color: #ecc94b; background: rgba(236,201,75,0.15); }
        .health-indicator.critical { color: #fc8181; background: rgba(252,129,129,0.15); }
        .header-right { color: #a0aec0; font-size: 13px; }

        .hero {
            background: linear-gradient(135deg, #1a202c 0%, #2d3748 100%);
            border-radius: 16px; padding: 40px 48px; margin-bottom: 24px;
            border: 1px solid #2d3748; text-align: center;
        }
        .hero-label { font-size: 14px; font-weight: 600; color: #a0aec0; text-transform: uppercase; letter-spacing: 1px; }
        .hero-value { font-size: 56px; font-weight: 800; color: #48bb78; margin: 8px 0; letter-spacing: -1px; }
        .hero-sub { font-size: 14px; color: #a0aec0; }
        .hero-sub span { color: #e2e8f0; font-weight: 600; }

        .cards {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 16px;
            margin-bottom: 24px;
        }
        .card {
            background: #1a202c; border-radius: 12px; padding: 20px 24px;
            border: 1px solid #2d3748; transition: all 0.2s;
        }
        .card:hover { border-color: #4299e1; transform: translateY(-2px); }
        .card-icon { font-size: 20px; margin-bottom: 8px; }
        #card-ebs .card-icon { color: #4299e1; }
        #card-eip .card-icon { color: #9f7aea; }
        #card-snapshot .card-icon { color: #ecc94b; }
        #card-rds .card-icon { color: #38b2ac; }
        .card-label { font-size: 12px; font-weight: 600; color: #a0aec0; text-transform: uppercase; letter-spacing: 0.5px; }
        .card-value { font-size: 24px; font-weight: 700; color: #e2e8f0; margin: 4px 0; }
        .card-count { font-size: 12px; color: #a0aec0; }

        .charts {
            display: grid;
            grid-template-columns: 2fr 1fr;
            gap: 16px;
            margin-bottom: 24px;
        }
        .chart-container {
            background: #1a202c; border-radius: 12px; padding: 20px 24px;
            border: 1px solid #2d3748; min-height: 200px;
        }
        .chart-container h3 {
            font-size: 14px; font-weight: 600; color: #a0aec0; margin-bottom: 16px;
        }
        .chart-container h3 i { margin-right: 8px; color: #4299e1; }
        .chart-container canvas { max-height: 250px; }

        .empty-state {
            display: flex; flex-direction: column; align-items: center;
            justify-content: center; padding: 40px 20px; color: #a0aec0; text-align: center;
        }
        .empty-state i { font-size: 48px; color: #2d3748; margin-bottom: 16px; }
        .empty-state p { font-size: 14px; max-width: 400px; }

        .activity {
            background: #1a202c; border-radius: 12px; padding: 20px 24px;
            border: 1px solid #2d3748; margin-bottom: 24px;
        }
        .activity h3 {
            font-size: 14px; font-weight: 600; color: #a0aec0; margin-bottom: 16px;
        }
        .activity h3 i { margin-right: 8px; color: #4299e1; }
        .table-wrapper { overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; font-size: 13px; }
        thead { background: #0f1419; }
        th { padding: 10px 12px; text-align: left; font-weight: 600; color: #a0aec0; border-bottom: 1px solid #2d3748; }
        td { padding: 10px 12px; border-bottom: 1px solid #2d3748; color: #e2e8f0; }
        tr:hover { background: #0f1419; }
        .status-deleted { color: #48bb78; }
        .status-quarantined { color: #ecc94b; }
        .status-stopped { color: #38b2ac; }
        .status-failed { color: #fc8181; }

        .footer { text-align: center; padding: 16px 0; color: #718096; font-size: 13px; }
        .footer .small { font-size: 11px; margin-top: 4px; }

        @media (max-width: 768px) {
            .cards { grid-template-columns: repeat(2, 1fr); }
            .charts { grid-template-columns: 1fr; }
            .hero-value { font-size: 36px; }
            .header { flex-direction: column; gap: 8px; text-align: center; }
        }
        @media (max-width: 480px) {
            .cards { grid-template-columns: 1fr; }
            .hero { padding: 24px 16px; }
            .hero-value { font-size: 28px; }
        }
    </style>
</head>
<body>
    <div class="dashboard">
        <header class="header">
            <div class="header-left">
                <h1><i class="fas fa-rocket"></i> Cloud FinOps Bot</h1>
                <span class="badge">v1.0</span>
            </div>
            <div class="header-center">
                <span class="health-indicator" id="healthStatus">
                    <i class="fas fa-circle"></i> Loading...
                </span>
            </div>
            <div class="header-right">
                <span id="lastUpdated">Loading...</span>
            </div>
        </header>

        <section class="hero">
            <div class="hero-content">
                <div class="hero-label">Total Monthly Savings</div>
                <div class="hero-value" id="totalSavings">$0.00</div>
                <div class="hero-sub">
                    <span id="resourceCount">0</span> resources deleted this month
                </div>
            </div>
        </section>

        <section class="cards">
            <div class="card" id="card-ebs">
                <div class="card-icon"><i class="fas fa-hdd"></i></div>
                <div class="card-label">EBS Volumes</div>
                <div class="card-value" id="savingsEBS">$0.00</div>
                <div class="card-count" id="countEBS">0 deleted</div>
            </div>
            <div class="card" id="card-eip">
                <div class="card-icon"><i class="fas fa-globe"></i></div>
                <div class="card-label">Elastic IPs</div>
                <div class="card-value" id="savingsEIP">$0.00</div>
                <div class="card-count" id="countEIP">0 released</div>
            </div>
            <div class="card" id="card-snapshot">
                <div class="card-icon"><i class="fas fa-camera"></i></div>
                <div class="card-label">Snapshots</div>
                <div class="card-value" id="savingsSnapshot">$0.00</div>
                <div class="card-count" id="countSnapshot">0 deleted</div>
            </div>
            <div class="card" id="card-rds">
                <div class="card-icon"><i class="fas fa-database"></i></div>
                <div class="card-label">RDS Instances</div>
                <div class="card-value" id="savingsRDS">$0.00</div>
                <div class="card-count" id="countRDS">0 stopped</div>
            </div>
        </section>

        <section class="charts">
            <div class="chart-container" id="trendChartContainer">
                <h3><i class="fas fa-chart-line"></i> Monthly Savings Trend</h3>
                <canvas id="savingsChart"></canvas>
            </div>
            <div class="chart-container" id="pieChartContainer">
                <h3><i class="fas fa-chart-pie"></i> Savings by Resource</h3>
                <canvas id="resourceChart"></canvas>
            </div>
        </section>

        <section class="activity">
            <h3><i class="fas fa-list"></i> Recent Activity</h3>
            <div class="table-wrapper">
                <table id="activityTable">
                    <thead>
                        <tr>
                            <th>Resource ID</th>
                            <th>Type</th>
                            <th>Savings</th>
                            <th>Action</th>
                            <th>Date</th>
                        </tr>
                    </thead>
                    <tbody id="activityBody"></tbody>
                </table>
            </div>
        </section>

        <footer class="footer">
            <p>Dashboard automatically updates daily from FinOps Bot</p>
            <p class="small">Data refreshed every 24 hours. All values are estimated monthly savings.</p>
        </footer>
    </div>

    <script>
        // Embedded data
        const dashboardData = %s;

        // Health indicator
        const healthEl = document.getElementById('healthStatus');
        if (dashboardData.healthStatus === 'healthy') {
            healthEl.className = 'health-indicator healthy';
            healthEl.innerHTML = '<i class="fas fa-circle"></i> System Healthy';
        } else if (dashboardData.healthStatus === 'warning') {
            healthEl.className = 'health-indicator warning';
            healthEl.innerHTML = '<i class="fas fa-circle"></i> Data Stale (24-48h)';
        } else {
            healthEl.className = 'health-indicator critical';
            healthEl.innerHTML = '<i class="fas fa-circle"></i> System Critical (No Data)';
        }

        document.getElementById('lastUpdated').textContent = 'Last updated: ' + dashboardData.lastUpdated;
        document.getElementById('totalSavings').textContent = '$' + dashboardData.totalSavings.toFixed(2);
        const totalResources = (dashboardData.breakdown.ebs.count || 0) +
                              (dashboardData.breakdown.eip.count || 0) +
                              (dashboardData.breakdown.snapshot.count || 0) +
                              (dashboardData.breakdown.rds.count || 0);
        document.getElementById('resourceCount').textContent = totalResources;

        document.getElementById('savingsEBS').textContent = '$' + (dashboardData.breakdown.ebs.savings || 0).toFixed(2);
        document.getElementById('countEBS').textContent = (dashboardData.breakdown.ebs.count || 0) + ' deleted';
        document.getElementById('savingsEIP').textContent = '$' + (dashboardData.breakdown.eip.savings || 0).toFixed(2);
        document.getElementById('countEIP').textContent = (dashboardData.breakdown.eip.count || 0) + ' released';
        document.getElementById('savingsSnapshot').textContent = '$' + (dashboardData.breakdown.snapshot.savings || 0).toFixed(2);
        document.getElementById('countSnapshot').textContent = (dashboardData.breakdown.snapshot.count || 0) + ' deleted';
        document.getElementById('savingsRDS').textContent = '$' + (dashboardData.breakdown.rds.savings || 0).toFixed(2);
        document.getElementById('countRDS').textContent = (dashboardData.breakdown.rds.count || 0) + ' stopped';

        // Activity table
        const tbody = document.getElementById('activityBody');
        if (dashboardData.recentActivity && dashboardData.recentActivity.length > 0) {
            dashboardData.recentActivity.forEach(item => {
                const tr = document.createElement('tr');
                const actionClass = 'status-' + item.action_taken.toLowerCase();
                tr.innerHTML =
                    '<td><code>' + item.resource_id + '</code></td>' +
                    '<td>' + item.resource_type + '</td>' +
                    '<td>$' + (item.estimated_savings || 0).toFixed(2) + '</td>' +
                    '<td><span class="' + actionClass + '">' + item.action_taken + '</span></td>' +
                    '<td>' + new Date(item.deletion_timestamp * 1000).toISOString().slice(0,10) + '</td>';
                tbody.appendChild(tr);
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#a0aec0;">No recent activity</td></tr>';
        }

        // Charts
        if (dashboardData.monthlyTrend && dashboardData.monthlyTrend.length > 0) {
            const ctx1 = document.getElementById('savingsChart').getContext('2d');
            new Chart(ctx1, {
                type: 'line',
                data: {
                    labels: dashboardData.monthlyTrend.map(d => d.month),
                    datasets: [{
                        label: 'Monthly Savings ($)',
                        data: dashboardData.monthlyTrend.map(d => d.savings),
                        borderColor: '#48bb78',
                        backgroundColor: 'rgba(72, 187, 120, 0.1)',
                        fill: true,
                        tension: 0.4,
                        pointBackgroundColor: '#48bb78',
                        pointBorderColor: '#1a202c',
                        pointBorderWidth: 2
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: { legend: { display: false } },
                    scales: {
                        y: { beginAtZero: true, grid: { color: 'rgba(160,174,192,0.1)' }, ticks: { callback: function(v) { return '$' + v; } } },
                        x: { grid: { display: false } }
                    }
                }
            });

            const ctx2 = document.getElementById('resourceChart').getContext('2d');
            new Chart(ctx2, {
                type: 'doughnut',
                data: {
                    labels: ['EBS Volumes', 'Elastic IPs', 'Snapshots', 'RDS Instances'],
                    datasets: [{
                        data: [
                            dashboardData.breakdown.ebs.savings || 0,
                            dashboardData.breakdown.eip.savings || 0,
                            dashboardData.breakdown.snapshot.savings || 0,
                            dashboardData.breakdown.rds.savings || 0
                        ],
                        backgroundColor: ['#4299e1', '#9f7aea', '#ecc94b', '#38b2ac'],
                        borderColor: '#1a202c',
                        borderWidth: 2
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { position: 'bottom', labels: { padding: 16, usePointStyle: true, pointStyle: 'circle' } }
                    }
                }
            });
        } else {
            document.getElementById('trendChartContainer').innerHTML =
                '<div class="empty-state"><i class="fas fa-info-circle"></i><p>No data available yet. The bot will start collecting data after its first run.</p></div>';
            document.getElementById('pieChartContainer').innerHTML =
                '<div class="empty-state"><i class="fas fa-info-circle"></i><p>No data available yet.</p></div>';
        }
    </script>
</body>
</html>`, jsonData)
}
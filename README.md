# Token Volume Tracker

A Go application for analyzing cryptocurrency trading volume data.

## Features

- Fetch historical trading volume data for any token
- Save data to CSV format with timestamps
- Analyze volume trends and patterns
- Generate summary statistics and visualizations
- Organized data storage with separate download and final directories

## Requirements

- Go 1.21 or higher

## Installation

1. Clone the repository (or download the code) to your Software Development directory:
```bash
cd "/Users/mikev/Library/Mobile Documents/com~apple~CloudDocs/Personal/Software Development"
git clone https://github.com/yourusername/token-volume-tracker.git "Token Volume Tracker"
```

2. Navigate to the project directory:
```bash
cd "Token Volume Tracker"
```

3. Download dependencies:
```bash
go mod download
```

4. Build the application:
```bash
go build -o token-volume-tracker cmd/main.go
```

## Directory Structure

The application uses the following directory structure:
```
Software Development/
├── Token Volume Tracker/         # Application directory
│   ├── cmd/                     # Command-line interface
│   ├── pkg/                     # Application packages
│   └── token-volume-tracker    # Compiled executable
└── Token Volume Tracker Data/   # Data directory
    ├── Download/               # Raw downloaded data
    └── Final/                  # Processed final data
```

## Usage

```bash
# Make sure you're in the Token Volume Tracker directory
cd "/Users/mikev/Library/Mobile Documents/com~apple~CloudDocs/Personal/Software Development/Token Volume Tracker"

# Fetch historical volume data
./token-volume-tracker fetch -token=CELO -days=365

# Analyze volume data
./token-volume-tracker analyze -input=Token\ Volume\ Tracker\ Data/Download/CELO_volume_2024-03-20_143022.csv -output=Token\ Volume\ Tracker\ Data/Final
```

### Available Commands

1. `fetch` - Fetch historical volume data
   - `-token` - Token symbol (e.g., CELO)
   - `-days` - Number of days of historical data to fetch (default: 7)

2. `analyze` - Analyze volume data (Coming Soon!)
   - `-input` - Input CSV file to analyze
   - `-output` - Output directory for analysis results

### Output Files

Downloaded data files are stored in the `Token Volume Tracker Data/Download` directory with the following naming convention:
```
{TOKEN}_volume_{YYYY-MM-DD}_{HHMMSS}.csv
```
Example: `CELO_volume_2024-03-20_143022.csv` (for CELO token downloaded on March 20, 2024 at 14:30:22)

The CSV files contain two columns:
```csv
Date,Volume (USD)
2024-03-20,1234567.89
2024-03-19,2345678.90
```

## Coming Soon

- Volume trend analysis
- Statistical summaries
- Data visualization
- Pattern detection
- Export to various formats

## License

MIT License 
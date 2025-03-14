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
│   ├── cmd/                      # Command-line interface
│   ├── pkg/                      # Application packages
│   ├── utils/                    # Utility scripts
│   └── token-volume-tracker      # Compiled executable
└── Token Volume Tracker Data/    # Data directory
    ├── Download/                 # Raw downloaded data
    └── Final/                    # Processed final data
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

2. `analyze` - Analyze volume data
   - `-input` - Input CSV file to analyze

### Output Files

#### Download Files
Raw data files are stored in the `Token Volume Tracker Data/Download` directory with the following naming conventions:
```
{TOKEN}_{START_DATE}-{END_DATE}_historical_data_coinmarketcap.csv for files downloaded from CoinMarketCap:
```
Example: `MAID_3_13_2024-3_13_2025_historical_data_coinmarketcap.csv`

```
{TOKEN}_usd-max.csv for files downloaded from CoinGecko:
```
Example: `QLC_usd-max.csv`

#### Analysis Files
Processed data files are stored in the `Token Volume Tracker Data/Final` directory with the following naming convention:
```
{TOKEN}_Trading_Average.csv
```
Example: `MAID_Trading_Average.csv`

The processed CSV files contain multiple columns including:
```csv
Name,Date,Volume,30DayAvg,90DayAvg,180DayAvg,LowVolumeDays30,LowVolumeDays90,LowVolumeDays180,HighestAvg30,HighestAvg90,HighestAvg180,ChangeFromHighAvg30%,ChangeFromHighAvg90%,ChangeFromHighAvg180%
```

#### Visualization Files
Excel charts with trading volume and 30-day rolling averages are generated from the CSV files and stored in the `Token Volume Tracker Data/Final` directory with the following naming convention:
```
{TOKEN}_Trading_Average.xlsx
```
Example: `MAID_Trading_Average.xlsx`

Each Excel file contains:
1. A "Chart" sheet with a visualization of trading volume and 30-day rolling average
2. A "Data" sheet with the complete analysis data

## Running the Analysis

1. Place historical data CSV files in the `Token Volume Tracker Data/Download` directory
2. Run the analysis command to process the data:
```bash
./token-volume-tracker analyze -input=Token\ Volume\ Tracker\ Data/Download/MAID_3_13_2024-3_13_2025_historical_data_coinmarketcap.csv -output=Token\ Volume\ Tracker\ Data/Final
```
3. Generate Excel charts from the CSV files:
   - First, copy the visualization script to the Final directory:
   ```bash
   cp utils/create_excel_charts.py "/Users/mikev/Library/Mobile Documents/com~apple~CloudDocs/Personal/Software Development/Token Volume Tracker Data/Final/"
   ```
   - Then run the script:
   ```bash
   cd "/Users/mikev/Library/Mobile Documents/com~apple~CloudDocs/Personal/Software Development/Token Volume Tracker Data/Final"
   python3 create_excel_charts.py
   ```

The script `create_excel_charts.py` will create professional Excel charts for each CSV file in the Final directory, with each Excel file containing:
1. A Chart sheet showing trading volume and 30-day rolling average
2. A Data sheet with the complete analysis data

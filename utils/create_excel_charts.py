#!/usr/bin/env python3
import os
import glob
import csv
import xlsxwriter
from datetime import datetime

def create_excel_with_chart(csv_file):
    # Use the same basename but with .xlsx extension.
    base_name = os.path.splitext(csv_file)[0]
    xlsx_file = base_name + '.xlsx'
    
    # Get token name from the CSV filename
    token_name = base_name.split('_')[0]
    
    # Create a new workbook and add a worksheet for the chart first
    workbook = xlsxwriter.Workbook(xlsx_file)
    
    # Add a chart sheet that will appear first
    chartsheet = workbook.add_chartsheet('Chart')
    
    # Add a data worksheet
    worksheet = workbook.add_worksheet("Data")
    
    # Create a date format for Excel
    date_format = workbook.add_format({'num_format': 'mm/dd/yy'})
    
    # Read the CSV file.
    with open(csv_file, 'r', newline='') as f:
        reader = csv.reader(f)
        data = list(reader)
    
    # Get the column indices
    header_row = data[0]
    try:
        date_col = header_row.index('Date')
    except ValueError:
        print(f"Warning: No 'Date' column found in {csv_file}")
        # Assume it's the second column (index 1)
        date_col = 1
    
    # Write headers to worksheet
    for col_num, cell in enumerate(data[0]):
        worksheet.write(0, col_num, cell)
    
    # Write data rows with proper date handling
    for row_num, row in enumerate(data[1:], 1):  # Skip header row
        for col_num, cell in enumerate(row):
            if col_num == date_col:
                # Convert date string to datetime object
                try:
                    # Parse yyyy-mm-dd format
                    date_value = datetime.strptime(cell, '%Y-%m-%d')
                    # Write as Excel date with formatting
                    worksheet.write_datetime(row_num, col_num, date_value, date_format)
                except ValueError:
                    # If date parsing fails, write as text
                    worksheet.write(row_num, col_num, cell)
            else:
                # Try to convert numerical values
                try:
                    # Convert to float if possible
                    cell_value = float(cell)
                    worksheet.write_number(row_num, col_num, cell_value)
                except ValueError:
                    # Not a number, write as string
                    worksheet.write(row_num, col_num, cell)
    
    # Create a line chart.
    chart = workbook.add_chart({'type': 'line'})
    
    # Set chart title and axis labels
    chart.set_title({
        'name': f'{token_name} Global Trading Volume and Rolling 30-Day Average',
        'name_font': {'size': 14, 'bold': True}
    })
    
    # Configure X-axis (date axis)
    chart.set_x_axis({
        'date_axis': True,
        'num_format': 'mm/dd/yy',
        'major_gridlines': {'visible': False},  # No vertical gridlines
        'line': {'color': 'black', 'width': 1},
    })
    
    # Configure Y-axis
    chart.set_y_axis({
        'name': 'Volume',
        'major_gridlines': {'visible': True, 'line': {'color': '#D9D9D9', 'width': 0.75}},
        'line': {'color': 'black', 'width': 1},
    })
    
    # Assume the first row is header.
    num_rows = len(data) - 1  # number of data rows (excluding header)
    
    # Get the column indices for Date, Volume, and 30DayAvg
    try:
        date_col = header_row.index('Date')
        volume_col = header_row.index('Volume')
        avg30_col = header_row.index('30DayAvg')
    except ValueError:
        print(f"Warning: Missing expected columns in {csv_file}")
        date_col = 1
        volume_col = 2
        avg30_col = 3
    
    # Excel column letters
    date_col_letter = chr(65 + date_col)  # A, B, C, etc.
    volume_col_letter = chr(65 + volume_col)
    avg30_col_letter = chr(65 + avg30_col)
    
    # Series for Volume
    chart.add_series({
        'name': '=Data!${0}$1'.format(volume_col_letter),
        'categories': '=Data!${0}$2:${0}${1}'.format(date_col_letter, num_rows+1),
        'values': '=Data!${0}$2:${0}${1}'.format(volume_col_letter, num_rows+1),
        'line': {
            'color': '#0F3D5E',  # Dark blue
            'width': 2,
            'smooth': False
        },
        'marker': {'type': 'none'},
    })
    
    # Series for 30-Day Average
    chart.add_series({
        'name': '=Data!${0}$1'.format(avg30_col_letter),
        'categories': '=Data!${0}$2:${0}${1}'.format(date_col_letter, num_rows+1),
        'values': '=Data!${0}$2:${0}${1}'.format(avg30_col_letter, num_rows+1),
        'line': {
            'color': '#ED7D31',  # Orange
            'width': 2,
            'smooth': False
        },
        'marker': {'type': 'none'},
    })
    
    # Configure legend
    chart.set_legend({'position': 'bottom'})
    
    # Set chart style
    chart.set_style(2)  # Use a cleaner built-in style
    
    # Make the chart larger
    chart.set_size({'width': 900, 'height': 500})
    
    # Remove chart border
    chart.set_plotarea({
        'border': {'none': True}
    })
    
    # Add the chart to the chartsheet
    chartsheet.set_chart(chart)
    
    workbook.close()
    print("Created:", xlsx_file)

def main():
    # Find all CSV files in the current directory.
    csv_files = glob.glob("*.csv")
    
    if not csv_files:
        print("No CSV files found in the current directory.")
        return
    
    for csv_file in csv_files:
        create_excel_with_chart(csv_file)

if __name__ == '__main__':
    main()
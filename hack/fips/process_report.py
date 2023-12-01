import json
import sys
import argparse
from datetime import datetime


def summarize_json(file_path):
    """Read and return JSON data from the given file."""
    try:
        with open(file_path, 'r') as file:
            return json.load(file)
    except Exception as e:
        print(f"An error occurred: {e}")
        sys.exit(1)


def parse_arguments():
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(description='Process report and generate HTML view.')
    parser.add_argument('file_path', help='Path to the JSON file to process')
    parser.add_argument('--fips', action='store_true', help='Filter to show only FIPS-compliant packages')
    return parser.parse_args()


def is_fips_compliant(entry):
    """Check if the entry is FIPS-compliant based on its annotations."""
    annotations = entry.get('csv', {}).get('metadata', {}).get('annotations', {})
    fips_compliant = annotations.get('features.operators.openshift.io/fips-compliant') == 'true'
    infrastructure_features = annotations.get('operators.openshift.io/infrastructure-features', "")
    fips_in_infrastructure = '"fips"' in infrastructure_features
    return fips_compliant or fips_in_infrastructure


def reorganize_data_by_package_name(data, fips_only):
    """Reorganize the 'Columns' data to group by 'packageName' and label with 'csv > metadata > name'."""
    grouped_data = {}

    for entry in data.get('Columns', []):
        if not fips_only or is_fips_compliant(entry):
            package_name = entry.get('packageName')
            bundle_name = entry.get('csv', {}).get('metadata', {}).get('name', 'Unknown')

            if package_name not in grouped_data:
                grouped_data[package_name] = {}

            grouped_data[package_name][bundle_name] = entry

    return grouped_data


def format_date(date_str):
    """Format the date from 'YYYY-MM-DD' to 'Month DD, YYYY'."""
    try:
        return datetime.strptime(date_str, '%Y-%m-%d').strftime('%b. %d, %Y')
    except ValueError:
        return 'Unknown Date'


def generate_report_note(args):
    """Generate a note for the report based on command-line arguments."""
    notes = []
    if args.fips:
        notes.append("filtered for claimed FIPS Compliance")
    # Additional flags can be handled here
    return " -- " + ", ".join(notes) if notes else ""


def determine_color(errors_warnings):
    """Determine the color based on error and warning content."""
    has_error = any("ERROR" in msg for msg in errors_warnings.get('errors', []))
    has_warning = any("Warning" in msg for msg in errors_warnings.get('warnings', []))

    if has_error:
        return "lightcoral"  # Light red for errors
    elif has_warning:
        return "lightyellow"  # Light orange for warnings
    return "lightgreen"  # Light green when no errors or warnings


def apply_color_to_data(data):
    """Apply color highlighting to the data based on errors and warnings."""
    for package_name, package_data in data.items():
        package_errors_warnings = {
            'errors': [],
            'warnings': []
        }
        for bundle_name, bundle_data in package_data.items():
            errors_warnings = bundle_data.get('errors', []) + bundle_data.get('warnings', [])
            color = determine_color({'errors': errors_warnings})
            bundle_data['color'] = color  # Apply color to the bundle
            package_errors_warnings['errors'].extend(bundle_data.get('errors', []))
            package_errors_warnings['warnings'].extend(bundle_data.get('warnings', []))

        # Determine and apply color to the package
        package_color = determine_color(package_errors_warnings)
        package_data['color'] = package_color


def json_to_html_report(flags, data, generated_at, report_note):
    """Convert JSON data to an HTML report."""
    apply_color_to_data(data)  # Apply color before generating the HTML
    formatted_date = format_date(generated_at)
    flags_rows = ''.join(f"<tr><td>{flag}</td><td>{value}</td></tr>" for flag, value in flags.items())
    formatted_json_data = json.dumps(data, indent=4)

    html_report = f"""
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Audit Report from {formatted_date}{report_note}</title>
        <script src="https://code.jquery.com/jquery-3.6.0.min.js"></script>
        <style>
            table {{
                width: 100%;
                border-collapse: collapse;
            }}
            table, th, td {{
                border: 1px solid black;
            }}
            th, td {{
                padding: 8px;
                text-align: left;
            }}
            th {{
                background-color: #f2f2f2;
            }}
            ul, #jsonViewer {{ list-style-type: none; }}
            .collapsible {{ cursor: pointer; }}
            .collapsed {{ display: none; }}
            .caret {{ cursor: pointer; user-select: none; }}
            .caret::before {{ content: "\\25B6"; color: black; display: inline-block; margin-right: 6px; }}
            .caret-down::before {{ content: "\\25BC"; }}
            .no-caret::before {{ content: ""; margin-right: 0; }}
        </style>
    </head>
    <body>
        <h1>Audit Report from {formatted_date}{report_note}</h1>
        <table>
            {flags_rows}
        </table>
        <p>Note: "errors" in the below report refers to static check-payload found FIPS errors. Packages with any bundle having these errors are shown with red below.</p>
        <ul id="jsonViewer"></ul>
        <script>
            function createList(container, obj) {{
                $.each(obj, function(key, value) {{
                    if (value === null || key === 'color') {{
                        // Skip null values and the 'color' key
                        return true; // 'true' is used in jQuery's each to continue the loop
                    }}
            
                    let li = $('<li>').appendTo(container);
                    let caretSpan = $('<span>').addClass('no-caret').appendTo(li);
                    let span = $('<span>').appendTo(li);
            
                    // Apply color if present and value is not null or an object itself
                    if (typeof value === 'object' && value !== null && value.color) {{
                        li.css('background-color', value.color);
                        // Remove the color property so it doesn't get displayed as a node
                        delete value.color;
                    }}
            
                    if (typeof value === 'object' && value !== null && (Array.isArray(value) || Object.keys(value).length > 0)) {{
                        caretSpan.removeClass('no-caret').addClass('caret');
                        caretSpan.click(function() {{
                            $(this).parent().find('> ul').toggle('fast');
                            $(this).toggleClass('caret-down');
                        }});
                        span.addClass('collapsible').text(key);
                        let ul = $('<ul>').addClass('collapsed').appendTo(li);
                        createList(ul, value);
                    }} else {{
                        // Handle non-object values (e.g., strings, numbers)
                        span.text(key + ': ' + value);
                    }}
                }});
            }}

            $(document).ready(function() {{
                createList($('#jsonViewer'), {formatted_json_data});
            }});
        </script>
    </body>
    </html>
    """
    return html_report


if __name__ == "__main__":
    args = parse_arguments()
    original_data = summarize_json(args.file_path)

    generated_at = original_data.get('GenerateAt', 'Unknown Date')
    flags = original_data.get('Flags', {})

    grouped_data = reorganize_data_by_package_name(original_data, args.fips)

    report_note = generate_report_note(args)
    html_report = json_to_html_report(flags, grouped_data, generated_at, report_note)

    report_file_path = "audit_report.html"

    with open(report_file_path, "w") as f:
        f.write(html_report)

    print(f"HTML report generated: {report_file_path}")

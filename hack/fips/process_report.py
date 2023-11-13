import json
import sys
import argparse


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

            # Label the entry with the 'csv > metadata > name', preserving all nested structures
            grouped_data[package_name][bundle_name] = entry

    return grouped_data


def json_to_html(data):
    """Convert JSON data to HTML for display."""
    html = """
    <!DOCTYPE html>
    <html>
    <head>
        <title>JSON Viewer</title>
        <script src="https://code.jquery.com/jquery-3.6.0.min.js"></script>
        <style>
            ul, #jsonViewer { list-style-type: none; }
            .collapsible { cursor: pointer; }
            .collapsed { display: none; }
            .caret { cursor: pointer; user-select: none; }
            .caret::before { content: "\\25B6"; color: black; display: inline-block; margin-right: 6px; }
            .caret-down::before { content: "\\25BC"; }
            .no-caret::before { content: ""; margin-right: 0; }
        </style>
    </head>
    <body>
        <ul id="jsonViewer"></ul>
        <script>
            function createList(container, obj) {
                $.each(obj, function(key, value) {
                    let li = $('<li>').appendTo(container);
                    let caretSpan = $('<span>').addClass('no-caret').appendTo(li);
                    let span = $('<span>').appendTo(li);

                    if (typeof value === 'object' && value !== null && (Array.isArray(value) || Object.keys(value).length > 0)) {
                        caretSpan.removeClass('no-caret').addClass('caret');
                        caretSpan.click(function() {
                            $(this).parent().find('> ul').toggle('fast');
                            $(this).toggleClass('caret-down');
                        });
                        span.addClass('collapsible').text(key);
                        let ul = $('<ul>').addClass('collapsed').appendTo(li);
                        createList(ul, value);
                    } else {
                        span.text(key + ': ' + value);
                    }
                });
            }

            $(document).ready(function() {
                createList($('#jsonViewer'), %s);
            });
        </script>
    </body>
    </html>
    """ % json.dumps(data, indent=4)
    return html


if __name__ == "__main__":
    args = parse_arguments()
    original_data = summarize_json(args.file_path)
    grouped_data = reorganize_data_by_package_name(original_data, args.fips)

    html_output = json_to_html(grouped_data)
    with open("../../json_viewer.html", "w") as f:
        f.write(html_output)

    print("HTML file generated: json_viewer.html")

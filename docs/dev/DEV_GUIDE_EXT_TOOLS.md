# External Tool Integration in the Audit Tool

This document showcases how external tools have been integrated into the Audit Tool. By examining these specific
implementations, developers can gain insights into adding similar integrations in the future.

## `operator-sdk` Integration

1. **Tool Purpose**: The `operator-sdk` tool assists in building, testing, and deploying Operator projects.
2. **Invocation in the Audit Tool**: The binary is invoked in the audit tool via system command calls. For instance:
   ```go
   cmd := exec.Command("operator-sdk", "bundle", "validate", "--select-optional", "suite=operatorframework")
   ```
3. **Handling Results**: Output and errors from the tool are captured and processed. For example:
   ```go
   output, err := cmd.CombinedOutput()
   if err != nil {
       log.Errorf("operator-sdk validation failed: %s", output)
   }
   ```

## `check-payload` Integration

1. **Tool Purpose**: The `check-payload` tool scans operator images to ensure compliance with specific standards.
2. **Invocation in the Audit Tool**: The binary is invoked similarly to the `operator-sdk`, but with different
   arguments:
   ```go
   cmd := exec.Command("/path/to/check-payload", "scan", "operator", "--spec", imageRef)
   ```
3. **Handling Results**: Output, warnings, and errors from this tool are captured and processed. Here's an example of
   how these results can be processed and incorporated into the audit tool's report:
   ```go
   output, err := cmd.CombinedOutput()
   if err != nil {
       // Handle error, potentially adding it to the report
   } else {
       // Process the output and distinguish between warnings and errors
       // Add warnings and errors to the report as appropriate
   }
   ```


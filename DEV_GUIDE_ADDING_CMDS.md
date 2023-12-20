# Commands and Sub-Commands Development Guide

This section of the development guide focuses on understanding, adding, and modifying commands and sub-commands within
the audit tool. By following the described patterns, developers can seamlessly introduce new functionalities.

## Adding a New Primary Command

1. **Define Your Command**:
   Begin by defining your command structure using the `cobra.Command` type, including:
   - `Use`: Command's name.
   - `Short`: A brief description.
   - `Long`: An extended description.

   Example from the `audit-tool`:
   ```go
   rootCmd := &cobra.Command{
       Use:   "audit-tool",
       Short: "An analytic tool to audit operator bundles and index catalogs",
       Long:  "The audit is an analytic tool which uses the Operator Framework solutions ...",
   }
   ```

2. **Add Sub-Commands (if necessary)**:
   For embedding sub-commands to your main command, employ the `AddCommand` method. As observed in the `audit-tool`,
   sub-commands like `index` and `custom` are integrated as:
   ```go
   rootCmd.AddCommand(index.NewCmd())
   rootCmd.AddCommand(custom.NewCmd())
   ```

3. **Execute the Command**:
   Ensure the primary command's execution in the `main` function:
   ```go
   if err := rootCmd.Execute(); err != nil {
       log.Fatal(err)
   }
   ```

## Tutorial: Add a Sub-Command to the `index` Command

The `index` command has sub-commands like `bundles` and `eus`. To introduce a new sub-command:

1. **Create a Sub-directory**:
   Organize by creating a sub-directory within `index`. Name it as per your sub-command. E.g., for a sub-command
   named `sample`, formulate a `sample` directory.

2. **Define Your Sub-Command**:
   In this directory, create a `command.go` file and define your sub-command structure:
   ```go
   package sample
   
   import (
       "github.com/spf13/cobra"
   )
   
   func NewCmd() *cobra.Command {
       cmd := &cobra.Command{
           Use:   "sample",
           Short: "Short description of sample",
           Long:  "Detailed description of sample...",
       }
       return cmd
   }
   ```

3. **Add Flags (optional)**:
   For flag additions to your sub-command, utilize the `Flags` method. For instance, to integrate a `--test` flag:
   ```go
   cmd.Flags().BoolP("test", "t", false, "Description of test flag")
   ```

4. **Integrate Sub-Command**:
   Navigate back to the `main.go` of `index` and add your new sub-command:
   ```go
   indexCmd.AddCommand(sample.NewCmd())
   ```

---

## Adding Flags with parameters to the `sample` Sub-Command

Flags offer flexibility to commands by allowing users to specify options or provide additional input. Here, we'll delve
into adding both boolean flags and flags that accept parameters to the `sample` sub-command.

1. **Boolean Flag**:
   A flag that signifies a simple `true` or `false` option.

   Example: Adding a `--test` flag to the `sample` sub-command:
   ```go
   cmd.Flags().BoolP("test", "t", false, "Description of test flag")
   ```
   Use: `audit-tool index sample --test`

2. **Flag with Parameter**:
   A flag that necessitates an accompanying value.

   Example: Introducing a `--input` flag which requires a string parameter:
   ```go
   cmd.Flags().StringP("input", "i", "", "Provide input data for the sample command")
   ```
   Use: `audit-tool index sample --input "This is sample input"`

3. **Utilize Flag Parameters in Command Logic**:
   To harness the values provided through flags, use the `cmd.Flag("flag-name").Value.String()` method.

   Example: Using the `--input` flag's value within the `sample` sub-command:
   ```go
   var input string = cmd.Flag("input").Value.String()
   if input != "" {
       fmt.Println("Received input:", input)
   } else {
       fmt.Println("No input provided.")
   }
   ```
   This code snippet checks if the `--input` flag has been provided a value, and if so, it prints the received input.

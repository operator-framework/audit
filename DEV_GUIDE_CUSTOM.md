## CUSTOM Directory Run Down

The `custom` directory focuses on providing specific functionalities that are tailored to the unique requirements of the
audit tool. These functionalities are organized as sub-commands, each having its own directory.

### NewCmd

This function initializes the overarching command for the 'custom' functionalities. It sets up the primary CLI structure
for the custom operations, guiding users to its specific sub-commands.

### Subdirectories:

Each subdirectory represents a specific custom functionality or operation. Here are the details:

#### validator

- **NewCmd**: This function sets up the command structure for the 'validator' functionality. It provides a brief
  description to the user about its purpose and defines available flags and options.
- **validation**: This function validates the provided flags and arguments specific to the 'validator' operation. It
  ensures that necessary inputs are present and correctly formatted.
- **run**: This function drives the core logic of the 'validator' functionality, making necessary calls to validate the
  data as per the defined criteria.

#### deprecate

- **NewCmd**: This function initializes the command for the 'deprecate' functionality, defining its purpose and
  available flags.
- **validation**: This function checks the provided flags and arguments to ensure they align with the 'deprecate'
  operation requirements.
- **run**: This function manages the 'deprecate' operation, handling the necessary steps to mark certain data as
  deprecated.

#### qa

- **NewCmd**: This function sets up the command for the 'qa' functionality, providing a brief description and defining
  the available flags.
- **validation**: This function validates user input, ensuring it matches the criteria set for the 'qa' operation.
- **run**: This function handles the 'qa' operation, performing quality assurance checks on the provided data.

#### multiarch

- **NewCmd**: This function initializes the command for the 'multiarch' functionality. It provides a description and
  lists available flags for the user.
- **validation**: This function checks the flags and arguments to ensure they fit the 'multiarch' operation's
  requirements.
- **run**: This function manages the 'multiarch' functionality, handling operations and checks related to multiple
  architectures.

# Features
Features are the place to put the services that do the business logic.


### How to create a new workspace
Workspace is the root of the projects (i.e. services). We need to create it once.
After workspace is created, we can create services inside it.

- Use `ronyup` cli to create a new workspace.
- Example: `ronyup setup workspace -d ./your-repo`


### How to create a new feature

- Use `ronyup` cli to create a new feature. Features could be one of the available templates: job, service or gateway
- Example: `ronyup setup feature -d ./your-repo -p auth-service`


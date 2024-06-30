# {{ .ProjectName }}

# Project Structure

| Folder           | Description                                                                            |
|------------------|----------------------------------------------------------------------------------------|
| cmd              | the executable/main packages are here                                                  |
| global           | global variables, constants, and helper functions that could be used by other packages |
| service          | business logic, service layer, and data access layer                                   |
| service/dto      | data transfer objects for service layer                                                |
| service/endpoint | endpoint handlers of the service                                                       |
| service/state    | state and related actions                                                              |


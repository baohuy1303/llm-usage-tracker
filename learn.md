- Go auto copies everything, so always be careful when handling references. Make sure that we are passing the address of the variable and not the value itself.

- In Express, you are used to the Database giving you a new object.
In Go, you are giving the Database an object and saying, "Hey, write the ID on this for me." Go way is significantly more memory-efficient because no matter how many layers the data travels through, you are only ever moving that one single "sticky note" with the memory address on it.

- For most funcs we only return an error object, because we are given the address, and by directly modifying the object on the address = we are already modifying the original object. This is why we don't need to return the object itself (no extra copying of data). Only error is returned.

- Use structs / create new struct when you need to add custom funcs to a type that's not defined in the same package. AND when you need to pass data between layers. So that we don't have to pass multiple arguments at the top layers down. Only pass it once at the top layer.
And overall just more clean when we need to change params.

- context.Context: When the user closes the tab, the ctx is "canceled." SQLite and Redis see this signal immediately and stop the query right then and there, saving your server resources. Every log message from the Service and Database folders can then look at the ctx and print that same ID.

Flow:
1. HTTP Handler receives a request. (create ctx and pass it down)
2. HTTP Handler calls the Service. (final error handling and response sending)
3. Service calls the Repository. (business logic)
4. Repository interacts with the Database. (data access)
5. Database returns the result to the Repository. (data access)
6. Goes back up the chain.

In Go, each layer returns (result, error).

The Handler is the final stop—it decides whether to log the error and what HTTP status code (400, 404, 500) to send back to the user.

Express equivalent:

app.listen() - http.ListenAndServe
Router - http.ServeMux
Middleware - func(http.Handler) http.Handler
Controller - ProjectHandler
Service - ProjectService
Repository - ProjectRepo

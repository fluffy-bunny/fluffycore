# Sessions.

This is a DI wrapper around gorilla sessions

## The stores are singletons.

backend store (there can only be one. Memory, redist, etc)
cookie store

## The sessions are transient

This is so that they can be instantiated and initialized.

## The Factory is scoped

This give us the request context.  
The factory is responsible for instantiating and storing a transient session object.
It is there to give out that session object to any other scoped object

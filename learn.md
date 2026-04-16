- Go auto copies everything, so always be careful when handling references. Make sure that we are passing the address of the variable and not the value itself.

- In Express, you are used to the Database giving you a new object.
In Go, you are giving the Database an object and saying, "Hey, write the ID on this for me." Go way is significantly more memory-efficient because no matter how many layers the data travels through, you are only ever moving that one single "sticky note" with the memory address on it.

- For most funcs we only return an error object, because we are given the address, and by directly modifying the object on the address = we are already modifying the original object. This is why we don't need to return the object itself (no extra copying of data). Only error is returned.
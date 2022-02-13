counter = 0

request = function()
   path = "/echo/" .. counter
   wrk.headers["X-Counter"] = counter
   counter = counter + 1
   return wrk.format(nil, path)
end
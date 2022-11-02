counter = 0

request = function()
	path = "/echo-post"
	counter = counter + 1
	wrk.method = "POST"
	wrk.body = '{"x": "something", "y": "anotherThing", "randomId": ' .. counter .. '}'
	wrk.headers["X-Counter"] = counter
	return wrk.format(nil, path)
end

response = function(status, headers, body)
	if status ~= 200 then
		io.write(body)
	end
end

package ucl

import (
	"net/http"
)

// PageRequest allows you to make a get call with a depth
type PageRequest struct {
	URL   string
	Depth int
}

// PageResponse will be returned on a call
type PageResponse struct {
	Request  *PageRequest
	Response *http.Response
	Err      error
}

// URLCaller can be used to make paralel url calls
type URLCaller struct {
	MaxCalls int
	requests []*PageRequest
	jobs     []chan *PageResponse
}

func (caller *URLCaller) tryStartNext() {
	// If we don't have urls we are don't have to start next
	if len(caller.requests) == 0 {
		return
	}

	// Check if we have room for another job if not we return
	if len(caller.jobs) >= caller.MaxCalls {
		return
	}

	job := make(chan *PageResponse)
	// Get the first value in the queue
	request := caller.requests[0]
	// Discard the first element
	caller.requests = caller.requests[1:]

	call := func(request *PageRequest) {
		res, err := http.Get(request.URL)

		job <- &PageResponse{
			Request:  request,
			Response: res,
			Err:      err,
		}
	}

	go call(request)
	caller.jobs = append(caller.jobs, job)
}

// Get enqueues a url call, the response can be retrieved with
func (caller *URLCaller) Get(request *PageRequest) {
	caller.requests = append(caller.requests, request)
	caller.tryStartNext()
}

// Next returns the next response that is completed, it will return nill if there are no more calls running
func (caller *URLCaller) Next() *PageResponse {
	jobAmount := len(caller.jobs)
	if len(caller.jobs) == 0 {
		return nil
	}

	var index int
	var res *PageResponse

	F:
	for i, job := range caller.jobs {
		select {
		case res = <-job:
			index = i
			// We found a completed job so we break out of the loop
			break F
		}
	}

	// We need to remove the job from the array
	caller.jobs[index] = caller.jobs[jobAmount-1]
	caller.jobs[jobAmount-1] = nil
	caller.jobs = caller.jobs[:jobAmount-1]

	// We will start a new job if there is something left in the queue
	caller.tryStartNext()

	return res
}

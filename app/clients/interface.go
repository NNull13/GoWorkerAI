package clients

import "GoWorkerAI/app/runtime"

type Interface interface {
	Subscribe(*runtime.Runtime)
}

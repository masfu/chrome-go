package chrome

// PageEvent represents a Chrome lifecycle event that can be waited on.
//
// Upstream equivalent: Page::DOM_CONTENT_LOADED, Page::LOAD, etc.
type PageEvent string

const (
	// EventDOMContentLoaded fires when the DOMContentLoaded event completes.
	EventDOMContentLoaded PageEvent = "DOMContentLoaded"
	// EventFirstContentfulPaint fires at the first contentful paint.
	EventFirstContentfulPaint PageEvent = "firstContentfulPaint"
	// EventFirstImagePaint fires at the first image paint.
	EventFirstImagePaint PageEvent = "firstImagePaint"
	// EventFirstMeaningfulPaint fires at the first meaningful paint.
	EventFirstMeaningfulPaint PageEvent = "firstMeaningfulPaint"
	// EventFirstPaint fires at the first paint.
	EventFirstPaint PageEvent = "firstPaint"
	// EventInit fires when the page starts loading.
	EventInit PageEvent = "init"
	// EventInteractiveTime fires when the page becomes interactive.
	EventInteractiveTime PageEvent = "interactiveTime"
	// EventLoad fires when the load event completes.
	EventLoad PageEvent = "load"
	// EventNetworkIdle fires when the network becomes idle.
	EventNetworkIdle PageEvent = "networkIdle"
)

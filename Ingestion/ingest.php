<?PHP
spl_autoload_register(function ($class_name) {
	include $class_name . '.php';
});

$start_time = $_SERVER["REQUEST_TIME_FLOAT"];

$body = file_get_contents('php://input');

if (!$body) {
	throw new Exception('INVALID REQUEST', ERROR_FORBIDDEN);
}

$request = new endpointRequest($start_time);
$request->populateRequest($body);
$request->generateRequests();
echo $request->enqueueRequests();
?>

<?PHP
include 'dataRequest.php';

define('REDIS_ADDRESS', "127.0.0.1:6379");
define('PENDING_QUEUE', "Pending");
define('VALUES_HASH', "Values");
define('STATS_HASH', "Stats");

class endpointRequest
{
    private $endpoint;
    private $data_requests;
    private $ready;
    private $has_requests;
    private $start_time;
    
    public function __construct($time)
    {
        $this->endpoint      = new stdClass;
        $this->ready         = false;
        $this->has_requests  = false;
        $this->data_requests = array();
        $this->start_time    = floatval($time) * 1000;
    }
    
    //This function accepts the raw post data and extracts
    //endpoint information and creates a dataRequest for
    //each set of values in the "data" object
    public function populateRequest($post_string)
    {
        parse_str($post_string, $post_variables);
        
        foreach ($post_variables as $field => $value) {
            switch ($field) {
                case "endpoint":
                    foreach ($value as $fld => $val) {
                        switch ($fld) {
                            case "method":
                                $this->endpoint->method = $val;
                                break;
                            case "url":
                                $this->endpoint->url = $val;
                                break;
                            default:
                                break;
                        }
                    }
                    break;
                case "data":
                    //Create new dataRequest for each set of values
                    foreach ($value as $data_point => $data_value) {
                        $req = new dataRequest($data_value);
                        $this->data_requests[$req->getUUID()] = $req;
                    }
                    break;
                default:
                    break;
            }
        }
        if (!empty($this->data_requests) && !empty($this->endpoint)) {
            $this->ready = true;
        }
        
    }
    
    //This function creats the final postback url
    //with data values substituted for each dataRequest
    public function generateRequests()
    {
        if ($this->ready) {
            foreach ($this->data_requests as $uuid => $request) {
                $request->setFormattedURL($this->endpoint->url);
            }
            
            if (!empty($this->data_requests)) {
                $this->has_requests = true;
            }
        }
    }
    
    //This function will handle all interactions between PHP and Redis
    //Each data request will be pushed to corresponding data structs in Redis
    public function enqueueRequests()
    {
        if (!$this->has_requests) {
		return "No requests to send";
	}
	
	$finished = "Error connecting to redis";
        
        try {
            $redis = new Redis();
            $redis->pconnect(REDIS_ADDRESS);
            
            $multi = $redis->multi();
            
            //Executing all posts to Redis in one multi batch
            foreach ($this->data_requests as $key => $request) {
                $uuid = $request->getUUID();
                
                $multi->lPush(PENDING_QUEUE, $uuid);
                $multi->hSet(VALUES_HASH, $uuid, $request->getFormattedURL());
                $multi->hSet(VALUES_HASH, $uuid . ':method', $this->endpoint->method);
                $multi->hSet(STATS_HASH, $uuid . ':start', $this->start_time);
            }
            
            $ret = $multi->exec();
            
            $finished = "Postback Delivered";
            
            //Seach results for any errors from Redis commands
            foreach ($ret as $idx => $result) {
                if (!$result) {
                    $finished = "Redis call failed: " . $idx;
                }
            }
            
        }
        catch (Exception $e) {
            return "Error posting to Redis";
        }
        
        return $finished;
        
    }
    
}

?>

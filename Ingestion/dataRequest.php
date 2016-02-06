<?PHP

//This class will be the base for all postback requests
//One instance of this class will be created for each
//Key-Value pair in the original post data
class dataRequest {
	private $sub_keys;
	private $sub_values;
	private $finalURL;
	private $uuid;
	
	public function __construct($data) {
		$this->uuid = uniqid('', false);
		$this->sub_keys = array();
		$this->sub_values = array();
		foreach($data as $key => $value) {
			$this->sub_keys[] = "{" . $key . "}";
			$this->sub_values[] = $value;
		}
	}
	
	public function getUUID() {
		return $this->uuid;
	}	
	public function getFormattedURL() {
		return $this->finalURL;
	}

	//This function substitutes values from data into the endpoint URL
	public function setFormattedURL($url_template) {
		$url =  str_replace($this->sub_keys, $this->sub_values, $url_template);
		$this->finalURL = preg_replace('/{(.*?)}/', '', $url);	
	}

}
?>

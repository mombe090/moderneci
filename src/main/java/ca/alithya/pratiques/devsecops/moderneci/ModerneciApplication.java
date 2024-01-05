package ca.alithya.pratiques.devsecops.moderneci;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;


@SpringBootApplication
@RestController
public class ModerneciApplication {

	public static void main(String[] args) {
		SpringApplication.run(ModerneciApplication.class, args);
	}

	@GetMapping("/ping")
	public ResponseEntity<String> pong() {
		return new ResponseEntity<String>("pong", null, 200);
	}	
}

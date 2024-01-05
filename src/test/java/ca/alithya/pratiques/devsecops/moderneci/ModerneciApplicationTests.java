package ca.alithya.pratiques.devsecops.moderneci;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;

import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;


@WebMvcTest
class ModerneciApplicationTests {

    @Autowired
    private MockMvc mockMvc;

    @Test
    public void testPong() throws Exception {
        mockMvc.perform(MockMvcRequestBuilders.get("/ping")
                .accept(MediaType.APPLICATION_JSON))
                .andExpect(status().isOk())
                .andExpect(content().string("pong"));
    }
	
}

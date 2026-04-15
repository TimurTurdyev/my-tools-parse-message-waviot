export namespace main {
	
	export class AuthStatusResponse {
	    status: string;
	    expiresAt: number;
	    loginUrl: string;
	    storedPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.expiresAt = source["expiresAt"];
	        this.loginUrl = source["loginUrl"];
	        this.storedPath = source["storedPath"];
	    }
	}
	export class JWTCaptureSession {
	    url: string;
	    nonce: string;
	    snippet: string;
	
	    static createFrom(source: any = {}) {
	        return new JWTCaptureSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.nonce = source["nonce"];
	        this.snippet = source["snippet"];
	    }
	}

}


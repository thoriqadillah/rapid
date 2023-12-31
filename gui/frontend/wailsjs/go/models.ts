export namespace client {
	
	export class Download {
	    id: string;
	    name: string;
	    url: string;
	    provider: string;
	    size: number;
	    type: string;
	    chunklen: number;
	    resumable: boolean;
	    progress: number;
	    expired: boolean;
	    downloadedChunks: number[];
	    timeLeft: number;
	    speed: number;
	    status: string;
	    // Go type: time
	    date: any;
	
	    static createFrom(source: any = {}) {
	        return new Download(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.provider = source["provider"];
	        this.size = source["size"];
	        this.type = source["type"];
	        this.chunklen = source["chunklen"];
	        this.resumable = source["resumable"];
	        this.progress = source["progress"];
	        this.expired = source["expired"];
	        this.downloadedChunks = source["downloadedChunks"];
	        this.timeLeft = source["timeLeft"];
	        this.speed = source["speed"];
	        this.status = source["status"];
	        this.date = this.convertValues(source["date"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Cookie {
	    name: string;
	    value: string;
	    path: string;
	    domain: string;
	    // Go type: time
	    expirationDate: any;
	    secure: boolean;
	    httpOnly: boolean;
	    sameSite: string;
	
	    static createFrom(source: any = {}) {
	        return new Cookie(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.value = source["value"];
	        this.path = source["path"];
	        this.domain = source["domain"];
	        this.expirationDate = this.convertValues(source["expirationDate"], null);
	        this.secure = source["secure"];
	        this.httpOnly = source["httpOnly"];
	        this.sameSite = source["sameSite"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Request {
	    url: string;
	    provider: string;
	    client?: string;
	    mimeType?: string;
	    userAgent?: string;
	    cookies?: Cookie[];
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.provider = source["provider"];
	        this.client = source["client"];
	        this.mimeType = source["mimeType"];
	        this.userAgent = source["userAgent"];
	        this.cookies = this.convertValues(source["cookies"], Cookie);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}


export namespace app {
	
	export class HostInfo {
	    id: string;
	    name: string;
	    address: string;
	    user: string;
	    auth: string;
	    keyPath: string;
	    hasPassword: boolean;
	    keyInVault: boolean;
	    hasPassphrase: boolean;
	    os: string;
	    osLike: string;
	
	    static createFrom(source: any = {}) {
	        return new HostInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.address = source["address"];
	        this.user = source["user"];
	        this.auth = source["auth"];
	        this.keyPath = source["keyPath"];
	        this.hasPassword = source["hasPassword"];
	        this.keyInVault = source["keyInVault"];
	        this.hasPassphrase = source["hasPassphrase"];
	        this.os = source["os"];
	        this.osLike = source["osLike"];
	    }
	}
	export class HostInput {
	    id: string;
	    name: string;
	    address: string;
	    user: string;
	    auth: string;
	    password: string;
	    keyPath: string;
	    importKey: boolean;
	    passphrase: string;
	    clearPassphrase: boolean;
	
	    static createFrom(source: any = {}) {
	        return new HostInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.address = source["address"];
	        this.user = source["user"];
	        this.auth = source["auth"];
	        this.password = source["password"];
	        this.keyPath = source["keyPath"];
	        this.importKey = source["importKey"];
	        this.passphrase = source["passphrase"];
	        this.clearPassphrase = source["clearPassphrase"];
	    }
	}

}

export namespace hostimport {
	
	export class Candidate {
	    alias: string;
	    address: string;
	    user: string;
	    keyPath: string;
	    keyStatus: string;
	    status: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.alias = source["alias"];
	        this.address = source["address"];
	        this.user = source["user"];
	        this.keyPath = source["keyPath"];
	        this.keyStatus = source["keyStatus"];
	        this.status = source["status"];
	        this.note = source["note"];
	    }
	}
	export class Preview {
	    path: string;
	    candidates: Candidate[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new Preview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.candidates = this.convertValues(source["candidates"], Candidate);
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
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

export namespace settings {
	
	export class Settings {
	    theme: string;
	    fontFamily: string;
	    fontSize: number;
	    lineHeight: number;
	    cursorStyle: string;
	    cursorBlink: boolean;
	    scrollback: number;
	    autoLockMinutes: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.fontFamily = source["fontFamily"];
	        this.fontSize = source["fontSize"];
	        this.lineHeight = source["lineHeight"];
	        this.cursorStyle = source["cursorStyle"];
	        this.cursorBlink = source["cursorBlink"];
	        this.scrollback = source["scrollback"];
	        this.autoLockMinutes = source["autoLockMinutes"];
	    }
	}

}


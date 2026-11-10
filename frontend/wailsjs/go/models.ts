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


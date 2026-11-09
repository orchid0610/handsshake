export namespace main {
	
	export class Host {
	    name: string;
	    address: string;
	    user: string;
	    keyPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Host(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.address = source["address"];
	        this.user = source["user"];
	        this.keyPath = source["keyPath"];
	    }
	}

}


type Sex = "Male" | "Female"

interface Role {
	id: number;
	name: string;
	gender: Sex;
}

interface Issue {
	id: number;
	name: string;
}

interface Tag {
	id: number;
	name: string;
	issues: Issue[];
}

interface User {
	id: number;
	name: string;
	age: number;
	discount: number;
	role_id: number;
	role: Role;
	tags: Tag[];
}


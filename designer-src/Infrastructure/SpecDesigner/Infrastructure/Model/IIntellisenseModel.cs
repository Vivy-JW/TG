using System;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x0200002C RID: 44
	public interface IIntellisenseModel
	{
		// Token: 0x17000056 RID: 86
		// (get) Token: 0x06000110 RID: 272
		// (set) Token: 0x06000111 RID: 273
		bool IsFavorited { get; set; }

		// Token: 0x17000057 RID: 87
		// (get) Token: 0x06000112 RID: 274
		string FullName { get; }

		// Token: 0x17000058 RID: 88
		// (get) Token: 0x06000113 RID: 275
		// (set) Token: 0x06000114 RID: 276
		string Name { get; set; }

		// Token: 0x17000059 RID: 89
		// (get) Token: 0x06000115 RID: 277
		// (set) Token: 0x06000116 RID: 278
		string Description { get; set; }

		// Token: 0x1700005A RID: 90
		// (get) Token: 0x06000117 RID: 279
		IntellisenseEnum Type { get; }

		// Token: 0x06000118 RID: 280
		string ToString();
	}
}

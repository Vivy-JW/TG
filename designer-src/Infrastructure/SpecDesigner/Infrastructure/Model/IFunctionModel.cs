using System;
using SpecDesignerCommon;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000023 RID: 35
	public interface IFunctionModel
	{
		// Token: 0x1700001E RID: 30
		// (get) Token: 0x06000075 RID: 117
		// (set) Token: 0x06000076 RID: 118
		PackageKey ProgramKey { get; set; }

		// Token: 0x1700001F RID: 31
		// (get) Token: 0x06000077 RID: 119
		// (set) Token: 0x06000078 RID: 120
		string Name { get; set; }

		// Token: 0x17000020 RID: 32
		// (get) Token: 0x06000079 RID: 121
		// (set) Token: 0x0600007A RID: 122
		string Description { get; set; }

		// Token: 0x17000021 RID: 33
		// (get) Token: 0x0600007B RID: 123
		// (set) Token: 0x0600007C RID: 124
		Scope Scope { get; set; }

		// Token: 0x17000022 RID: 34
		// (get) Token: 0x0600007D RID: 125
		// (set) Token: 0x0600007E RID: 126
		int SortIndex { get; set; }

		// Token: 0x0600007F RID: 127
		void Clear();
	}
}

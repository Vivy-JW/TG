using System;

namespace SpecDesigner.Infrastructure.Extension
{
	// Token: 0x02000038 RID: 56
	public interface IDragable
	{
		// Token: 0x17000072 RID: 114
		// (get) Token: 0x06000156 RID: 342
		Type DragType { get; }

		// Token: 0x17000073 RID: 115
		// (get) Token: 0x06000157 RID: 343
		bool CanDrag { get; }
	}
}

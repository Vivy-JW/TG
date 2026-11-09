using System;
using SpecDesigner.CodeEditWindow.View;

namespace SpecDesigner.CodeEditWindow
{
	// Token: 0x02000026 RID: 38
	public interface IComparisonWindow
	{
		// Token: 0x17000027 RID: 39
		// (get) Token: 0x06000152 RID: 338
		DiffTextViewer SourceViewer { get; }

		// Token: 0x17000028 RID: 40
		// (get) Token: 0x06000153 RID: 339
		DiffTextViewer TargetViewer { get; }
	}
}

using System;
using DifferenceEngine;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000040 RID: 64
	public class DiffResult
	{
		// Token: 0x17000067 RID: 103
		// (get) Token: 0x060002CF RID: 719 RVA: 0x000189EF File Offset: 0x00016BEF
		// (set) Token: 0x060002D0 RID: 720 RVA: 0x000189F7 File Offset: 0x00016BF7
		public int LineNumber { get; set; }

		// Token: 0x17000068 RID: 104
		// (get) Token: 0x060002D1 RID: 721 RVA: 0x00018A00 File Offset: 0x00016C00
		// (set) Token: 0x060002D2 RID: 722 RVA: 0x00018A08 File Offset: 0x00016C08
		public DiffResultSpanStatus Type { get; set; }

		// Token: 0x060002D3 RID: 723 RVA: 0x00018A11 File Offset: 0x00016C11
		public DiffResult(int lineNumber, DiffResultSpanStatus type)
		{
			this.LineNumber = lineNumber;
			this.Type = type;
		}
	}
}

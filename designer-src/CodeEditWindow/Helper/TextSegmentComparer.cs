using System;
using System.Collections.Generic;
using ICSharpCode.AvalonEdit.Document;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000024 RID: 36
	public class TextSegmentComparer : IComparer<TextSegment>
	{
		// Token: 0x06000145 RID: 325 RVA: 0x0000C2EC File Offset: 0x0000A4EC
		public int Compare(TextSegment x, TextSegment y)
		{
			if (x.StartOffset == y.StartOffset)
			{
				return x.EndOffset.CompareTo(y.EndOffset);
			}
			return x.StartOffset.CompareTo(y.StartOffset);
		}
	}
}

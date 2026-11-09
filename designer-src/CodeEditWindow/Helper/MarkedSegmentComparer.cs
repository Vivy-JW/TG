using System;
using System.Collections.Generic;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000023 RID: 35
	public class MarkedSegmentComparer : IComparer<MarkedSegment>
	{
		// Token: 0x06000143 RID: 323 RVA: 0x0000C270 File Offset: 0x0000A470
		public int Compare(MarkedSegment x, MarkedSegment y)
		{
			if (x.StartLine == y.StartLine)
			{
				return 0;
			}
			if (x.StartLine.Offset == y.StartLine.Offset)
			{
				return x.StartLine.EndOffset.CompareTo(y.StartLine.EndOffset);
			}
			return x.StartLine.Offset.CompareTo(y.StartLine.Offset);
		}
	}
}

using System;

namespace DifferenceEngine
{
	// Token: 0x02000002 RID: 2
	public class TextLine : IComparable
	{
		// Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00001050
		public TextLine(string str)
		{
			this.Line = str.Replace("\t", "    ");
			this._hash = str.GetHashCode();
		}

		// Token: 0x06000002 RID: 2 RVA: 0x0000207A File Offset: 0x0000107A
		public int CompareTo(object obj)
		{
			return this._hash.CompareTo(((TextLine)obj)._hash);
		}

		// Token: 0x04000001 RID: 1
		public string Line;

		// Token: 0x04000002 RID: 2
		public int _hash;
	}
}

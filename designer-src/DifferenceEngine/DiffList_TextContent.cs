using System;
using System.Collections;

namespace DifferenceEngine
{
	// Token: 0x02000005 RID: 5
	public class DiffList_TextContent : IDiffList
	{
		// Token: 0x0600000A RID: 10 RVA: 0x0000218C File Offset: 0x0000118C
		public DiffList_TextContent(string content)
		{
			this._lines = new ArrayList();
			foreach (string text in content.Split(new string[]
			{
				Environment.NewLine,
				"\n"
			}, StringSplitOptions.RemoveEmptyEntries))
			{
				this._lines.Add(new TextLine(text));
			}
		}

		// Token: 0x0600000B RID: 11 RVA: 0x000021EE File Offset: 0x000011EE
		public int Count()
		{
			return this._lines.Count;
		}

		// Token: 0x0600000C RID: 12 RVA: 0x000021FB File Offset: 0x000011FB
		public IComparable GetByIndex(int index)
		{
			return (TextLine)this._lines[index];
		}

		// Token: 0x04000006 RID: 6
		private const int MaxLineLength = 1024;

		// Token: 0x04000007 RID: 7
		private ArrayList _lines;
	}
}

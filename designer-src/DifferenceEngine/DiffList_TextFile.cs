using System;
using System.Collections;

namespace DifferenceEngine
{
	// Token: 0x02000004 RID: 4
	public class DiffList_TextFile : IDiffList
	{
		// Token: 0x06000005 RID: 5 RVA: 0x00002094 File Offset: 0x00001094
		public DiffList_TextFile(string fileName)
		{
			this._lines = new ArrayList();
			this._oriSource = new ArrayList();
			string[] array = fileName.Split(new char[] { '\n' });
			for (int i = 0; i < array.Length; i++)
			{
				if (!string.IsNullOrEmpty(array[i].Trim()))
				{
					this._lines.Add(new TextLine(array[i].Replace("\r", "")));
				}
				else
				{
					this._lines.Add(new TextLine(array[i].Replace("\r", "").Trim().ToLower()));
				}
				this._oriSource.Add(array[i]);
			}
		}

		// Token: 0x17000001 RID: 1
		// (get) Token: 0x06000006 RID: 6 RVA: 0x0000214F File Offset: 0x0000114F
		public ArrayList OriSource
		{
			get
			{
				return this._oriSource;
			}
		}

		// Token: 0x06000007 RID: 7 RVA: 0x00002157 File Offset: 0x00001157
		public int Count()
		{
			return this._lines.Count;
		}

		// Token: 0x06000008 RID: 8 RVA: 0x00002164 File Offset: 0x00001164
		public IComparable GetByIndex(int index)
		{
			return (TextLine)this._lines[index];
		}

		// Token: 0x06000009 RID: 9 RVA: 0x00002177 File Offset: 0x00001177
		public ArrayList Add(TextLine text)
		{
			this._lines.Add(text);
			return this._lines;
		}

		// Token: 0x04000003 RID: 3
		private const int MaxLineLength = 1024;

		// Token: 0x04000004 RID: 4
		private ArrayList _lines;

		// Token: 0x04000005 RID: 5
		private ArrayList _oriSource;
	}
}

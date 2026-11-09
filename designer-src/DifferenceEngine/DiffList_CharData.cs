using System;

namespace DifferenceEngine
{
	// Token: 0x02000007 RID: 7
	public class DiffList_CharData : IDiffList
	{
		// Token: 0x06000010 RID: 16 RVA: 0x0000229D File Offset: 0x0000129D
		public DiffList_CharData(string charData)
		{
			this._charList = charData.ToCharArray();
		}

		// Token: 0x06000011 RID: 17 RVA: 0x000022B1 File Offset: 0x000012B1
		public int Count()
		{
			return this._charList.Length;
		}

		// Token: 0x06000012 RID: 18 RVA: 0x000022BB File Offset: 0x000012BB
		public IComparable GetByIndex(int index)
		{
			return this._charList[index];
		}

		// Token: 0x04000009 RID: 9
		private char[] _charList;
	}
}

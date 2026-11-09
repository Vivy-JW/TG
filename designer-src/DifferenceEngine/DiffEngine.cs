using System;
using System.Collections;

namespace DifferenceEngine
{
	// Token: 0x02000009 RID: 9
	public class DiffEngine
	{
		// Token: 0x06000013 RID: 19 RVA: 0x000022CA File Offset: 0x000012CA
		public DiffEngine()
		{
			this._source = null;
			this._dest = null;
			this._matchList = null;
			this._stateList = null;
			this._level = DiffEngineLevel.FastImperfect;
		}

		// Token: 0x06000014 RID: 20 RVA: 0x000022F8 File Offset: 0x000012F8
		private int GetSourceMatchLength(int destIndex, int sourceIndex, int maxLength)
		{
			int num = 0;
			while (num < maxLength && this._dest.GetByIndex(destIndex + num).CompareTo(this._source.GetByIndex(sourceIndex + num)) == 0)
			{
				num++;
			}
			return num;
		}

		// Token: 0x06000015 RID: 21 RVA: 0x00002338 File Offset: 0x00001338
		private void GetLongestSourceMatch(DiffState curItem, int destIndex, int destEnd, int sourceStart, int sourceEnd)
		{
			int num = destEnd - destIndex + 1;
			int num2 = 0;
			int num3 = -1;
			for (int i = sourceStart; i <= sourceEnd; i++)
			{
				int num4 = Math.Min(num, sourceEnd - i + 1);
				if (num4 <= num2)
				{
					break;
				}
				int sourceMatchLength = this.GetSourceMatchLength(destIndex, i, num4);
				if (sourceMatchLength > num2)
				{
					num3 = i;
					num2 = sourceMatchLength;
				}
				i += num2;
			}
			if (num3 == -1)
			{
				curItem.SetNoMatch();
				return;
			}
			curItem.SetMatch(num3, num2);
		}

		// Token: 0x06000016 RID: 22 RVA: 0x000023A8 File Offset: 0x000013A8
		private void ProcessRange(int destStart, int destEnd, int sourceStart, int sourceEnd)
		{
			int num = -1;
			int num2 = -1;
			DiffState diffState = null;
			for (int i = destStart; i <= destEnd; i++)
			{
				int num3 = destEnd - i + 1;
				if (num3 <= num2)
				{
					break;
				}
				DiffState byIndex = this._stateList.GetByIndex(i);
				if (!byIndex.HasValidLength(sourceStart, sourceEnd, num3))
				{
					this.GetLongestSourceMatch(byIndex, i, destEnd, sourceStart, sourceEnd);
				}
				if (byIndex.Status == DiffStatus.Matched)
				{
					switch (this._level)
					{
					case DiffEngineLevel.FastImperfect:
						if (byIndex.Length > num2)
						{
							num = i;
							num2 = byIndex.Length;
							diffState = byIndex;
						}
						i += byIndex.Length - 1;
						break;
					case DiffEngineLevel.Medium:
						if (byIndex.Length > num2)
						{
							num = i;
							num2 = byIndex.Length;
							diffState = byIndex;
							i += byIndex.Length - 1;
						}
						break;
					default:
						if (byIndex.Length > num2)
						{
							num = i;
							num2 = byIndex.Length;
							diffState = byIndex;
						}
						break;
					}
				}
			}
			if (num < 0)
			{
				return;
			}
			int startIndex = diffState.StartIndex;
			this._matchList.Add(DiffResultSpan.CreateNoChange(num, startIndex, num2));
			if (destStart < num && sourceStart < startIndex)
			{
				this.ProcessRange(destStart, num - 1, sourceStart, startIndex - 1);
			}
			int num4 = num + num2;
			int num5 = startIndex + num2;
			if (destEnd > num4 && sourceEnd > num5)
			{
				this.ProcessRange(num4, destEnd, num5, sourceEnd);
			}
		}

		// Token: 0x06000017 RID: 23 RVA: 0x000024EB File Offset: 0x000014EB
		public double ProcessDiff(IDiffList source, IDiffList destination, DiffEngineLevel level)
		{
			this._level = level;
			return this.ProcessDiff(source, destination);
		}

		// Token: 0x06000018 RID: 24 RVA: 0x000024FC File Offset: 0x000014FC
		public double ProcessDiff(IDiffList source, IDiffList destination)
		{
			DateTime now = DateTime.Now;
			this._source = source;
			this._dest = destination;
			this._matchList = new ArrayList();
			int num = this._dest.Count();
			int num2 = this._source.Count();
			if (num > 0 && num2 > 0)
			{
				this._stateList = new DiffStateList(num);
				this.ProcessRange(0, num - 1, 0, num2 - 1);
			}
			return (DateTime.Now - now).TotalSeconds;
		}

		// Token: 0x06000019 RID: 25 RVA: 0x00002578 File Offset: 0x00001578
		private bool AddChanges(ArrayList report, int curDest, int nextDest, int curSource, int nextSource)
		{
			bool flag = false;
			int num = nextDest - curDest;
			int num2 = nextSource - curSource;
			if (num > 0)
			{
				if (num2 > 0)
				{
					int num3 = Math.Min(num, num2);
					report.Add(DiffResultSpan.CreateReplace(curDest, curSource, num3));
					if (num > num2)
					{
						curDest += num3;
						report.Add(DiffResultSpan.CreateAddDestination(curDest, num - num2));
					}
					else if (num2 > num)
					{
						curSource += num3;
						report.Add(DiffResultSpan.CreateDeleteSource(curSource, num2 - num));
					}
				}
				else
				{
					report.Add(DiffResultSpan.CreateAddDestination(curDest, num));
				}
				flag = true;
			}
			else if (num2 > 0)
			{
				report.Add(DiffResultSpan.CreateDeleteSource(curSource, num2));
				flag = true;
			}
			return flag;
		}

		// Token: 0x0600001A RID: 26 RVA: 0x00002614 File Offset: 0x00001614
		public ArrayList DiffReport()
		{
			ArrayList arrayList = new ArrayList();
			int num = this._dest.Count();
			int num2 = this._source.Count();
			if (num == 0)
			{
				if (num2 > 0)
				{
					arrayList.Add(DiffResultSpan.CreateDeleteSource(0, num2));
				}
				return arrayList;
			}
			if (num2 == 0)
			{
				arrayList.Add(DiffResultSpan.CreateAddDestination(0, num));
				return arrayList;
			}
			this._matchList.Sort();
			int num3 = 0;
			int num4 = 0;
			DiffResultSpan diffResultSpan = null;
			foreach (object obj in this._matchList)
			{
				DiffResultSpan diffResultSpan2 = (DiffResultSpan)obj;
				if (!this.AddChanges(arrayList, num3, diffResultSpan2.DestIndex, num4, diffResultSpan2.SourceIndex) && diffResultSpan != null)
				{
					diffResultSpan.AddLength(diffResultSpan2.Length);
				}
				else
				{
					arrayList.Add(diffResultSpan2);
				}
				num3 = diffResultSpan2.DestIndex + diffResultSpan2.Length;
				num4 = diffResultSpan2.SourceIndex + diffResultSpan2.Length;
				diffResultSpan = diffResultSpan2;
			}
			this.AddChanges(arrayList, num3, num, num4, num2);
			return arrayList;
		}

		// Token: 0x0400000E RID: 14
		private IDiffList _source;

		// Token: 0x0400000F RID: 15
		private IDiffList _dest;

		// Token: 0x04000010 RID: 16
		private ArrayList _matchList;

		// Token: 0x04000011 RID: 17
		private DiffEngineLevel _level;

		// Token: 0x04000012 RID: 18
		private DiffStateList _stateList;
	}
}

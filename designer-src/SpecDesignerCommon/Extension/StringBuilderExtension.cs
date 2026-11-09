using System;
using System.Text;

namespace SpecDesignerCommon.Extension
{
	// Token: 0x02000127 RID: 295
	public static class StringBuilderExtension
	{
		// Token: 0x06000A4E RID: 2638 RVA: 0x0003376F File Offset: 0x0003196F
		public static string Substring(this StringBuilder sb, int startIndex, int length)
		{
			return sb.ToString(startIndex, length);
		}

		// Token: 0x06000A4F RID: 2639 RVA: 0x0003377C File Offset: 0x0003197C
		public static StringBuilder Remove(this StringBuilder sb, char ch)
		{
			int i = 0;
			while (i < sb.Length)
			{
				if (sb[i] == ch)
				{
					sb.Remove(i, 1);
				}
				else
				{
					i++;
				}
			}
			return sb;
		}

		// Token: 0x06000A50 RID: 2640 RVA: 0x000337B0 File Offset: 0x000319B0
		public static StringBuilder RemoveFromEnd(this StringBuilder sb, int num)
		{
			return sb.Remove(sb.Length - num, num);
		}

		// Token: 0x06000A51 RID: 2641 RVA: 0x000337C1 File Offset: 0x000319C1
		public static void Clear(this StringBuilder sb)
		{
			sb.Length = 0;
		}

		// Token: 0x06000A52 RID: 2642 RVA: 0x000337CC File Offset: 0x000319CC
		public static StringBuilder LTrim(this StringBuilder sb)
		{
			if (sb.Length != 0)
			{
				int num = 0;
				int length = sb.Length;
				while (sb[num] == ' ' && num < length)
				{
					num++;
				}
				if (num > 0)
				{
					sb.Remove(0, num);
				}
			}
			return sb;
		}

		// Token: 0x06000A53 RID: 2643 RVA: 0x00033810 File Offset: 0x00031A10
		public static StringBuilder RTrim(this StringBuilder sb)
		{
			if (sb.Length != 0)
			{
				int length = sb.Length;
				int num = length - 1;
				while (sb[num] == ' ' && num > -1)
				{
					num--;
				}
				if (num < length - 1)
				{
					sb.Remove(num + 1, length - num - 1);
				}
			}
			return sb;
		}

		// Token: 0x06000A54 RID: 2644 RVA: 0x0003385C File Offset: 0x00031A5C
		public static StringBuilder Trim(this StringBuilder sb)
		{
			if (sb.Length != 0)
			{
				int num = 0;
				int num2 = sb.Length;
				while (sb[num] == ' ' && num < num2)
				{
					num++;
				}
				if (num > 0)
				{
					sb.Remove(0, num);
					num2 = sb.Length;
				}
				num = num2 - 1;
				while (sb[num] == ' ' && num > -1)
				{
					num--;
				}
				if (num < num2 - 1)
				{
					sb.Remove(num + 1, num2 - num - 1);
				}
			}
			return sb;
		}

		// Token: 0x06000A55 RID: 2645 RVA: 0x000338D2 File Offset: 0x00031AD2
		public static int IndexOf(this StringBuilder sb, char value)
		{
			return sb.IndexOf(value, 0);
		}

		// Token: 0x06000A56 RID: 2646 RVA: 0x000338DC File Offset: 0x00031ADC
		public static int IndexOf(this StringBuilder sb, char value, int startIndex)
		{
			for (int i = startIndex; i < sb.Length; i++)
			{
				if (sb[i] == value)
				{
					return i;
				}
			}
			return -1;
		}

		// Token: 0x06000A57 RID: 2647 RVA: 0x00033907 File Offset: 0x00031B07
		public static int IndexOf(this StringBuilder sb, string value)
		{
			return sb.IndexOf(value, 0, false);
		}

		// Token: 0x06000A58 RID: 2648 RVA: 0x00033912 File Offset: 0x00031B12
		public static int IndexOf(this StringBuilder sb, string value, int startIndex)
		{
			return sb.IndexOf(value, startIndex, false);
		}

		// Token: 0x06000A59 RID: 2649 RVA: 0x0003391D File Offset: 0x00031B1D
		public static int IndexOf(this StringBuilder sb, string value, bool ignoreCase)
		{
			return sb.IndexOf(value, 0, ignoreCase);
		}

		// Token: 0x06000A5A RID: 2650 RVA: 0x00033928 File Offset: 0x00031B28
		public static int IndexOf(this StringBuilder sb, string value, int startIndex, bool ignoreCase)
		{
			int length = value.Length;
			int num = sb.Length - length + 1;
			if (!ignoreCase)
			{
				for (int i = startIndex; i < num; i++)
				{
					if (sb[i] == value[0])
					{
						int num2 = 1;
						while (num2 < length && sb[i + num2] == value[num2])
						{
							num2++;
						}
						if (num2 == length)
						{
							return i;
						}
					}
				}
			}
			else
			{
				for (int j = startIndex; j < num; j++)
				{
					if (char.ToLower(sb[j]) == char.ToLower(value[0]))
					{
						int num2 = 1;
						while (num2 < length && char.ToLower(sb[j + num2]) == char.ToLower(value[num2]))
						{
							num2++;
						}
						if (num2 == length)
						{
							return j;
						}
					}
				}
			}
			return -1;
		}

		// Token: 0x06000A5B RID: 2651 RVA: 0x000339E8 File Offset: 0x00031BE8
		public static bool StartsWith(this StringBuilder sb, string value)
		{
			return sb.StartsWith(value, 0, false);
		}

		// Token: 0x06000A5C RID: 2652 RVA: 0x000339F3 File Offset: 0x00031BF3
		public static bool StartsWith(this StringBuilder sb, string value, bool ignoreCase)
		{
			return sb.StartsWith(value, 0, ignoreCase);
		}

		// Token: 0x06000A5D RID: 2653 RVA: 0x00033A00 File Offset: 0x00031C00
		public static bool StartsWith(this StringBuilder sb, string value, int startIndex, bool ignoreCase)
		{
			int length = value.Length;
			int num = startIndex + length;
			if (!ignoreCase)
			{
				for (int i = startIndex; i < num; i++)
				{
					if (sb[i] != value[i - startIndex])
					{
						return false;
					}
				}
			}
			else
			{
				for (int j = startIndex; j < num; j++)
				{
					if (char.ToLower(sb[j]) != char.ToLower(value[j - startIndex]))
					{
						return false;
					}
				}
			}
			return true;
		}
	}
}

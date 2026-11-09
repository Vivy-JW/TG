using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Text;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x0200000B RID: 11
	public partial class ErrorCheckWindow : Window
	{
		// Token: 0x06000059 RID: 89 RVA: 0x00005090 File Offset: 0x00003290
		public ErrorCheckWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x0600005A RID: 90 RVA: 0x000050D4 File Offset: 0x000032D4
		public ErrorCheckWindow(CodeTextEditor editor)
			: this()
		{
			base.Title = Application.Current.FindResource("menu_ProgramErrorCheck") as string;
			string[] array = editor.Text.Split(new char[] { '\n' });
			List<ErrorList> list = new List<ErrorList>();
			Regex regex = new Regex("^ *IF", RegexOptions.IgnoreCase | RegexOptions.Multiline);
			Regex regex2 = new Regex("^ *END *IF", RegexOptions.IgnoreCase | RegexOptions.Multiline);
			Regex regex3 = new Regex("END *IF", RegexOptions.IgnoreCase | RegexOptions.Multiline);
			int num = 0;
			for (int i = 0; i < array.Length; i++)
			{
				if (regex.IsMatch(array[i]))
				{
					ErrorList errorList = new ErrorList();
					errorList.check = false;
					errorList.head = "IF";
					errorList.h_line = i + 1;
					errorList.h_blankspace = array[i].IndexOf("IF");
					if (regex3.IsMatch(array[i]))
					{
						errorList.tail = "END IF";
						errorList.t_line = i + 1;
						errorList.t_blankspace = 999;
					}
					list.Add(errorList);
					if (num == 0)
					{
						num = i + 1;
					}
				}
				if (regex2.IsMatch(array[i]))
				{
					List<ErrorList> list2 = list.Where<ErrorList>((ErrorList a) => !a.check && a.tail == null).ToList<ErrorList>();
					if (list2[list2.Count<ErrorList>() - 1].h_blankspace == array[i].IndexOf("END IF"))
					{
						list2[list2.Count<ErrorList>() - 1].tail = "END IF";
						list2[list2.Count<ErrorList>() - 1].t_line = i + 1;
						list2[list2.Count<ErrorList>() - 1].t_blankspace = array[i].IndexOf("END IF");
						if (list2[list2.Count<ErrorList>() - 1].h_line == num)
						{
							num = 0;
						}
					}
					else
					{
						bool flag = false;
						foreach (ErrorList errorList2 in list2)
						{
							if (errorList2.h_blankspace == array[i].IndexOf("END IF"))
							{
								errorList2.tail = "END IF";
								errorList2.t_line = i + 1;
								errorList2.t_blankspace = array[i].IndexOf("END IF");
								if (errorList2.h_line == num)
								{
									num = 0;
								}
								flag = true;
								break;
							}
						}
						if (!flag)
						{
							list.Add(new ErrorList
							{
								check = false,
								tail = "END IF",
								t_line = i + 1,
								t_blankspace = array[i].IndexOf("END IF")
							});
						}
					}
					List<ErrorList> list3 = list.Where<ErrorList>((ErrorList a) => !a.check).ToList<ErrorList>();
					if (num == 0)
					{
						foreach (ErrorList errorList3 in list3)
						{
							errorList3.check = true;
						}
					}
				}
			}
			List<ErrorList> list4 = list.Where<ErrorList>((ErrorList a) => a.head == null || a.tail == null).ToList<ErrorList>();
			StringBuilder stringBuilder = new StringBuilder();
			string text = string.Empty;
			foreach (ErrorList errorList4 in list4)
			{
				if (errorList4.tail == null)
				{
					text = (Application.Current.FindResource("Message_CannotFindEndIF") as string) + "\r\n";
					stringBuilder.Append(string.Format(text, errorList4.h_line));
				}
				else
				{
					text = (Application.Current.FindResource("Message_CannotFindIF") as string) + "\r\n";
					stringBuilder.Append(string.Format(text, errorList4.t_line));
				}
			}
			if (string.IsNullOrEmpty(stringBuilder.ToString()))
			{
				this.txtErrorMag.Text = Application.Current.FindResource("Message_KeyWordNotNull") as string;
				return;
			}
			this.txtErrorMag.Text = stringBuilder.ToString();
		}
	}
}
